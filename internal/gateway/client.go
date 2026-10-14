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

// Client represents a single active, bidirectional WebSocket connection.
type Client struct {
	Hub       *Hub
	UserID    uuid.UUID
	SessionID uuid.UUID
	Conn      *websocket.Conn
	Send      chan []byte

	ctx       context.Context
	ctxCancel context.CancelFunc
	closeOnce sync.Once
	wg        sync.WaitGroup
}

func (c *Client) GetUserID() uuid.UUID    { return c.UserID }
func (c *Client) GetSessionID() uuid.UUID { return c.SessionID }

// NewClient initializes a Client instance with its own isolated cancellation context.
func NewClient(ctx context.Context, hub *Hub, userID, sessionID uuid.UUID, conn *websocket.Conn) *Client {
	clientCtx, cancel := context.WithCancel(ctx)
	return &Client{
		Hub:       hub,
		UserID:    userID,
		SessionID: sessionID,
		Conn:      conn,
		Send:      make(chan []byte, sendBufferLength),
		ctx:       clientCtx,
		ctxCancel: cancel,
	}
}

// Close gracefully cancels the client context and terminates the network connection.
func (c *Client) Close() {
	c.closeOnce.Do(func() {
		c.ctxCancel()
		if err := c.Conn.Close(); err != nil {
			slog.DebugContext(c.ctx, "failed to close websocket connection cleanly",
				"user_id", c.UserID,
				"session_id", c.SessionID,
				"error", err,
			)
		}
	})
}

// StartPumps launches the background read and write loops for the client connection.
func (c *Client) StartPumps() {
	c.wg.Add(2)

	go func() {
		defer c.wg.Done()
		c.writePump()
	}()

	go func() {
		defer c.wg.Done()
		c.readPump()
	}()
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
	if err := c.Conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
		slog.DebugContext(c.ctx, "failed to set write deadline for close frame",
			"user_id", c.UserID,
			"session_id", c.SessionID,
			"error", err,
		)
		return
	}

	if err := c.Conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")); err != nil {
		slog.DebugContext(c.ctx, "failed to write websocket close frame",
			"user_id", c.UserID,
			"session_id", c.SessionID,
			"error", err,
		)
	}
}

// flushMessageBatch writes the initial message and drains any additional queued messages into a slice.
func (c *Client) flushMessageBatch(firstMsg []byte) error {
	if err := c.Conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
		slog.ErrorContext(c.ctx, "failed to set write deadline for message batch",
			"user_id", c.UserID,
			"session_id", c.SessionID,
			"error", err,
		)
		return err
	}

	w, err := c.Conn.NextWriter(websocket.TextMessage)
	if err != nil {
		slog.ErrorContext(c.ctx, "failed to acquire next WebSocket text writer",
			"user_id", c.UserID,
			"session_id", c.SessionID,
			"error", err,
		)
		return err
	}

	messages := [][]byte{firstMsg}

drainLoop:
	for {
		select {
		case msg, ok := <-c.Send:
			if !ok {
				if closeErr := w.Close(); closeErr != nil {
					slog.DebugContext(c.ctx, "failed to close writer on send channel close",
						"user_id", c.UserID,
						"session_id", c.SessionID,
						"error", closeErr,
					)
				}
				return fmt.Errorf("send channel closed during batch drain")
			}
			messages = append(messages, msg)
		default:
			break drainLoop
		}
	}

	for i, msg := range messages {
		if i > 0 {
			if _, err := w.Write([]byte{'\n'}); err != nil {
				if closeErr := w.Close(); closeErr != nil {
					slog.DebugContext(c.ctx, "failed to close writer after newline write error",
						"user_id", c.UserID,
						"session_id", c.SessionID,
						"error", closeErr,
					)
				}
				return err
			}
		}
		if _, err := w.Write(msg); err != nil {
			if closeErr := w.Close(); closeErr != nil {
				slog.DebugContext(c.ctx, "failed to close writer after batch write error",
					"user_id", c.UserID,
					"session_id", c.SessionID,
					"error", closeErr,
				)
			}
			return err
		}
	}

	if err := w.Close(); err != nil {
		slog.ErrorContext(c.ctx, "failed to close WebSocket message writer",
			"user_id", c.UserID,
			"session_id", c.SessionID,
			"error", err,
		)
		return err
	}

	return nil
}

// writePing issues a WebSocket Control Ping frame with a write deadline.
func (c *Client) writePing() error {
	if err := c.Conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
		slog.ErrorContext(c.ctx, "failed to set write deadline for ping",
			"user_id", c.UserID,
			"session_id", c.SessionID,
			"error", err,
		)
		return err
	}

	if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
		slog.ErrorContext(c.ctx, "failed to write ping message",
			"user_id", c.UserID,
			"session_id", c.SessionID,
			"error", err,
		)
		return err
	}

	return nil
}

// readPump receives incoming frames, refreshes read deadlines, and dispatches handlers.
func (c *Client) readPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	if err := c.Conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		slog.ErrorContext(c.ctx, "failed to set initial read deadline",
			"user_id", c.UserID,
			"session_id", c.SessionID,
			"error", err,
		)
		return
	}

	c.Conn.SetPongHandler(func(string) error {
		if err := c.Conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
			slog.ErrorContext(c.ctx, "failed to refresh read deadline on pong",
				"user_id", c.UserID,
				"session_id", c.SessionID,
				"error", err,
			)
			return err
		}
		return nil
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

		c.handleMessage(msg)
	}
}

// handleMessage parses inbound message envelopes and dispatches registered event handlers.
func (c *Client) handleMessage(rawMsg []byte) {
	var msg Message
	if err := json.Unmarshal(rawMsg, &msg); err != nil {
		slog.WarnContext(c.ctx, "failed to unmarshal websocket message",
			"error", err,
			"user_id", c.UserID,
			"session_id", c.SessionID,
		)
		return
	}

	if msg.Type == "" {
		slog.WarnContext(c.ctx, "websocket message missing type field",
			"user_id", c.UserID,
			"session_id", c.SessionID,
		)
		return
	}

	handler, exists := c.Hub.GetHandler(msg.Type)
	if !exists {
		slog.WarnContext(c.ctx, "unregistered websocket event type",
			"type", msg.Type,
			"user_id", c.UserID,
			"session_id", c.SessionID,
		)
		return
	}

	go c.executeHandler(handler, msg)
}

// executeHandler executes an event handler with timeout context controls and panic safety guarantees.
func (c *Client) executeHandler(h MessageHandler, msg Message) {
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
