package gateway

import (
	"bonfire-api/internal/message"
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type WSMessage struct {
	Type string          `json:"t"`
	Data json.RawMessage `json:"d"`
}

type Client struct {
	UserID   uuid.UUID
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

func (c *Client) readPump(ctx context.Context, hub *Hub) {
	defer func() {
		hub.unregister <- c
		c.Close()
	}()

	c.Conn.SetReadLimit(512)
	_ = c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))

	c.Conn.SetPongHandler(func(string) error {
		_ = c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
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
		case "JOIN_CHANNEL":
			var data struct {
				ChannelID uuid.UUID `json:"channel_id"`
			}
			json.Unmarshal(wsMsg.Data, &data)
			hub.JoinRoom(data.ChannelID, c.UserID)

		case "LEAVE_CHANNEL":
			var data struct {
				ChannelID uuid.UUID `json:"channel_id"`
			}
			json.Unmarshal(wsMsg.Data, &data)
			hub.LeaveRoom(data.ChannelID, c.UserID)

		case "SEND_MESSAGE":
			var data message.SendReq
			if err := json.Unmarshal(wsMsg.Data, &data); err != nil {
				continue
			}

			// 1. Persist message to DB
			msgView, err := hub.msgService.PostMessage(ctx, c.UserID, data)
			if err != nil {
				continue
			}

			// 2. Broadcast to room
			payload, _ := json.Marshal(map[string]interface{}{
				"t": "NEW_MESSAGE",
				"d": msgView,
			})
			hub.BroadcastToRoom(data.ChannelID, payload)
		}
	}
}

func (c *Client) writePump() {
	for message := range c.Send {
		_ = c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
			break
		}
	}
	c.Close()
}
