package gateway

import (
	"context"
	"net/http"
	"time"

	"bonfire-api/internal/apperr"
	"bonfire-api/internal/httpio"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Replace with strict origin domain checks matching your config profiles
		return true
	},
}

type Handler struct {
	hub *Hub
}

func NewHandler(hub *Hub) *Handler {
	return &Handler{hub: hub}
}

func (h *Handler) ServeWS(w http.ResponseWriter, r *http.Request) error {
	userID, err := httpio.GetCtxUserID(r.Context())
	if err != nil {
		return apperr.NewUnauthorized(err)
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return err
	}

	client := &Client{
		UserID: userID,
		Conn:   conn,
		Send:   make(chan []byte, 256),
	}

	h.hub.register <- client

	// Handle standard async read/write loops natively
	go client.writePump()
	go client.readPump(r.Context(), h.hub)

	return nil
}

func (c *Client) writePump() {
	defer c.Conn.Close()
	for message := range c.Send {
		_ = c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
			return
		}
	}
}

func (c *Client) readPump(ctx context.Context, hub *Hub) {
	defer func() {
		hub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(512) // Restrict body payload sizes strictly
	_ = c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))

	// Configure connection to reset deadlines upon successful pong verifications
	c.Conn.SetPongHandler(func(string) error {
		_ = c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		_ = hub.cache.Heartbeat(ctx, c.UserID.String()) // Keep state green in Redis cache automatically
		return nil
	})

	for {
		_, msg, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}

		// If client prefers signaling updates over WebSocket wires rather than standard REST,
		// parse incoming client text frames here.
		_ = msg
	}
}
