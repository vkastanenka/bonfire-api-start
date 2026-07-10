package gateway

import (
	"bonfire-api/internal/outbox"
	"bonfire-api/internal/presence.go"
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	pingPeriod = 20 * time.Second
	pongWait   = 30 * time.Second
	writeWait  = 10 * time.Second
)

type WSMessage struct {
	Type string          `json:"t"`
	Data json.RawMessage `json:"d"`
}

type Client struct {
	UserID   uuid.UUID
	Presence presence.Presence
	Conn     *websocket.Conn
	Send     chan []byte
	isClosed bool
	mu       sync.Mutex
}

func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.isClosed {
		c.isClosed = true
		_ = c.Conn.Close()
	}
}

func (c *Client) readPump(hub *Hub) {
	defer func() {
		hub.unregister <- c
		c.Close()
	}()

	c.Conn.SetReadLimit(512)
	_ = c.Conn.SetReadDeadline(time.Now().Add(pongWait))

	c.Conn.SetPongHandler(func(string) error {
		_ = c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, msg, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}

		var wsMsg WSMessage
		if err := json.Unmarshal(msg, &wsMsg); err != nil {
			continue
		}

		switch wsMsg.Type {
		case "UPDATE_PRESENCE":
			var data struct {
				Presence string `json:"presence"`
			}
			if err := json.Unmarshal(wsMsg.Data, &data); err == nil {
				presenceEnum := presence.ParsePresence(data.Presence)
				c.Presence = presenceEnum

				err := func() error {
					bgCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
					defer cancel()

					return outbox.EmitPresenceUpdated(bgCtx, hub.store, outbox.PresenceUpdatedPayload{
						UserID:   c.UserID.String(),
						Presence: presenceEnum.String(),
					})
				}()

				if err != nil {
				}
			}
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
