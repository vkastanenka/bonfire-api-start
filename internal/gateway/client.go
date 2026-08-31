package gateway

import (
	"bonfire-api/internal/fields"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Network and buffer configuration governing connection lifecycles.
const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 30 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = 20 * time.Second

	// Maximum message size allowed from peer in bytes (4 KB).
	maxMessageSize = 4096

	// Channel buffer capacity for outgoing messages per client.
	sendBufferLength = 256

	// Timeout for processing individual incoming message handlers.
	handlerTimeout = 5 * time.Second
)

// WSMessage defines the JSON wire format for incoming and outgoing gateway frames.
// Ex: {"t": "chat.speak", "d": {"text": "Hello world"}}
type WSMessage struct {
	Type string          `json:"t"`
	Data json.RawMessage `json:"d"`
}

// Client represents a single active, bidirectional WebSocket connection.
type Client struct {
	ID        uuid.UUID
	UserID    fields.ID
	SessionID fields.ID
	Conn      *websocket.Conn // The underlying TCP WebSocket connection handle.
	Send      chan []byte     // Buffered channel for queuing outbound messages.

	ctx       context.Context
	cancelCtx context.CancelFunc
	closeOnce sync.Once
}

// NewClient initializes a Client instance.
func NewClient(ctx context.Context, userID, sessionID fields.ID, conn *websocket.Conn) *Client {
	clientCtx, cancel := context.WithCancel(ctx)
	return &Client{
		ID:        uuid.New(),
		UserID:    userID,
		SessionID: sessionID,
		Conn:      conn,
		Send:      make(chan []byte, sendBufferLength),
		ctx:       clientCtx,
		cancelCtx: cancel,
	}
}

// Close gracefully terminates the context, and closes the Send channel with the WS connection.
func (c *Client) Close() {
	c.closeOnce.Do(func() {
		c.cancelCtx()
		_ = c.Conn.Close()
		close(c.Send)
	})
}

// StartPumps launches the read and write loops for the client instance.
func (c *Client) StartPumps(hub *Hub) {
	go c.writePump()
	go c.readPump(hub)
}

// writePump handles outbound network operations: sending queued messages from the hub,
// batch-flushing buffered messages and transmitting periodic ping heartbeats.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Close()
	}()

	for {
		select {
		case <-c.ctx.Done():
			// Parent or connection context canceled; terminate loop.
			return

		case message, ok := <-c.Send:
			// Outbound message available on Send channel.
			if !ok {
				// The hub closed the channel; gracefully issue a WS close control frame.
				_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
				_ = c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.flushMessageBatch(message); err != nil {
				return
			}

		case <-ticker.C:
			// Heartbeat ticker fired; send WS ping frame to client.
			if err := c.writePing(); err != nil {
				return
			}
		}
	}
}

// flushMessageBatch writes the initial message frame and drains any additional messages sitting
// in the channel buffer into a single combined WebSocket network frame separated by newlines.
func (c *Client) flushMessageBatch(firstMsg []byte) error {
	if err := c.Conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
		return err
	}

	// Acquire a writer stream for a text frame.
	w, err := c.Conn.NextWriter(websocket.TextMessage)
	if err != nil {
		return err
	}

	// Write the primary message.
	if _, err := w.Write(firstMsg); err != nil {
		_ = w.Close()
		return err
	}

	// Drain any remaining messages already queued in c.Send.
	n := len(c.Send)
	for i := 0; i < n; i++ {
		msg, ok := <-c.Send
		if !ok {
			// Channel closed mid-batch; close current frame writer and signal error.
			_ = w.Close()
			return fmt.Errorf("send channel closed during batch drain")
		}
		if _, err := w.Write([]byte{'\n'}); err != nil {
			_ = w.Close()
			return err
		}
		if _, err := w.Write(msg); err != nil {
			_ = w.Close()
			return err
		}
	}

	return w.Close()
}

// writePing issues a low-level WebSocket Control Ping frame with a write deadline.
func (c *Client) writePing() error {
	if err := c.Conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
		return err
	}
	return c.Conn.WriteMessage(websocket.PingMessage, nil)
}

// readPump receives incoming WebSocket frames from the client connection, manages read deadlines,
// processes heartbeats, and routes valid frames to registered central hub event handlers.
func (c *Client) readPump(hub *Hub) {
	defer func() {
		// Guarantee unregistration executes regardless of context cancellation.
		hub.unregister <- c
		c.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	_ = c.Conn.SetReadDeadline(time.Now().Add(pongWait))

	// Reset read deadline every time a Pong reply is received from the client.
	c.Conn.SetPongHandler(func(string) error {
		return c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		_, msg, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.ErrorContext(c.ctx, "websocket read error", "user_id", c.UserID, "error", err)
			}
			break
		}

		c.dispatchFrame(hub, msg)
	}
}

// dispatchFrame parses inbound frame envelopes and dispatches registered event handlers.
func (c *Client) dispatchFrame(hub *Hub, rawMsg []byte) {
	var wsMsg WSMessage
	if err := json.Unmarshal(rawMsg, &wsMsg); err != nil || wsMsg.Type == "" {
		return
	}

	handler, exists := hub.GetHandler(wsMsg.Type)
	if !exists {
		slog.WarnContext(c.ctx, "unregistered websocket event type", "type", wsMsg.Type, "user_id", c.UserID)
		return
	}

	// Option A (Default): Synchronous execution prevents runaway goroutine spawn per client
	c.executeHandler(handler, wsMsg)
}

// executeHandler executes an event handler with timeout context controls and panic safety guarantees.
func (c *Client) executeHandler(h MessageHandler, msg WSMessage) {
	// Protect the application runtime against unhandled runtime panics inside event handlers.
	defer func() {
		if r := recover(); r != nil {
			slog.ErrorContext(c.ctx, "recovered panic in websocket handler",
				"type", msg.Type,
				"user_id", c.UserID,
				"panic", r,
			)
		}
	}()

	// Apply isolation context with enforced execution timeout budget.
	ctx, cancel := context.WithTimeout(c.ctx, handlerTimeout)
	defer cancel()

	if err := h(ctx, c, msg.Data); err != nil {
		slog.ErrorContext(ctx, "handler execution failed",
			"type", msg.Type,
			"user_id", c.UserID,
			"error", err,
		)
	}
}
