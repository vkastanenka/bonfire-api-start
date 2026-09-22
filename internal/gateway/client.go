package gateway

import (
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
type WSMessage struct {
	Type string          `json:"t"`
	Data json.RawMessage `json:"d,omitempty"`
}

// Client represents a single active, bidirectional WebSocket connection.
type Client struct {
	NodeID    uuid.UUID
	UserID    uuid.UUID
	SessionID uuid.UUID
	Conn      *websocket.Conn
	Send      chan []byte

	ctx       context.Context
	cancelCtx context.CancelFunc
	closeOnce sync.Once
	wg        sync.WaitGroup
}

func (c *Client) GetNodeID() uuid.UUID    { return c.NodeID }
func (c *Client) GetUserID() uuid.UUID    { return c.UserID }
func (c *Client) GetSessionID() uuid.UUID { return c.SessionID }

// NewClient initializes a Client instance with its own isolated cancellation context.
func NewClient(ctx context.Context, nodeID, userID, sessionID uuid.UUID, conn *websocket.Conn) *Client {
	clientCtx, cancel := context.WithCancel(ctx)
	return &Client{
		UserID:    userID,
		SessionID: sessionID,
		NodeID:    nodeID,
		Conn:      conn,
		Send:      make(chan []byte, sendBufferLength),
		ctx:       clientCtx,
		cancelCtx: cancel,
	}
}

// Close gracefully cancels the client context and terminates the network connection.
func (c *Client) Close() {
	c.closeOnce.Do(func() {
		c.cancelCtx()
		_ = c.Conn.Close()
	})
}

// StartPumps launches the background read and write loops for the client connection.
func (c *Client) StartPumps(hub *Hub) {
	c.wg.Add(2)

	go func() {
		defer c.wg.Done()
		c.writePump()
	}()

	go func() {
		defer c.wg.Done()
		c.readPump(hub)
	}()
}

// Wait blocks until both read and write pumps have terminated.
func (c *Client) Wait() {
	c.wg.Wait()
}

// writePump handles outbound network operations, batching, and heartbeats.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Close()
	}()

	for {
		select {
		case <-c.ctx.Done():
			// Send a WS close control frame using a detached context deadline guarantee
			c.sendCloseFrame()
			return

		case message, ok := <-c.Send:
			if !ok {
				c.sendCloseFrame()
				return
			}
			if err := c.flushMessageBatch(message); err != nil {
				return
			}

		case <-ticker.C:
			if err := c.writePing(); err != nil {
				return
			}
		}
	}
}

// sendCloseFrame attempts to write a standard WebSocket close control frame before teardown.
func (c *Client) sendCloseFrame() {
	_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
	_ = c.Conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
}

// flushMessageBatch writes the initial message and drains available queued messages.
func (c *Client) flushMessageBatch(firstMsg []byte) error {
	if err := c.Conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
		return err
	}

	w, err := c.Conn.NextWriter(websocket.TextMessage)
	if err != nil {
		return err
	}

	if _, err := w.Write(firstMsg); err != nil {
		_ = w.Close()
		return err
	}

	n := len(c.Send)

DrainLoop:
	for i := 0; i < n; i++ {
		select {
		case msg, ok := <-c.Send:
			if !ok {
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
		default:
			break DrainLoop
		}
	}

	return w.Close()
}

// writePing issues a WebSocket Control Ping frame with a write deadline.
func (c *Client) writePing() error {
	if err := c.Conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
		return err
	}
	return c.Conn.WriteMessage(websocket.PingMessage, nil)
}

// readPump receives incoming frames, refreshes read deadlines, and dispatches handlers.
func (c *Client) readPump(hub *Hub) {
	defer func() {
		hub.unregister <- c
		c.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	_ = c.Conn.SetReadDeadline(time.Now().Add(pongWait))

	c.Conn.SetPongHandler(func(string) error {
		return c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		_, msg, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.ErrorContext(c.ctx, "websocket read error",
					"user_id", c.UserID,
					"session_id", c.SessionID,
					"error", err,
				)
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
		slog.WarnContext(c.ctx, "unregistered websocket event type",
			"type", wsMsg.Type,
			"user_id", c.UserID,
			"session_id", c.SessionID,
		)
		return
	}

	go c.executeHandler(handler, wsMsg)
}

// executeHandler executes an event handler with timeout context controls and panic safety guarantees.
func (c *Client) executeHandler(h MessageHandler, msg WSMessage) {
	defer func() {
		if r := recover(); r != nil {
			slog.ErrorContext(c.ctx, "recovered panic in websocket handler",
				"type", msg.Type,
				"user_id", c.UserID,
				"session_id", c.SessionID,
				"panic", r,
			)
		}
	}()

	// Detach execution context from direct parent cancellation using context.WithoutCancel,
	// while still applying an isolated execution timeout budget.
	execCtx, cancel := context.WithTimeout(context.WithoutCancel(c.ctx), handlerTimeout)
	defer cancel()

	if err := h(execCtx, c, msg.Data); err != nil {
		slog.ErrorContext(execCtx, "websocket handler execution failed",
			"type", msg.Type,
			"user_id", c.UserID,
			"session_id", c.SessionID,
			"error", err,
		)
	}
}
