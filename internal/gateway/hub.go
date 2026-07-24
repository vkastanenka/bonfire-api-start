package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"bonfire-api/internal/cache"
	"bonfire-api/internal/db"

	"github.com/google/uuid"
)

// MessageHandler defines the signature for processing incoming client messages.
type MessageHandler func(ctx context.Context, client *Client, data json.RawMessage) error

type Hub struct {
	clients    map[uuid.UUID]*Client
	register   chan *Client
	unregister chan *Client
	handlers   map[string]MessageHandler // <--- Added handler registry
	mu         sync.RWMutex

	store db.Store
	cache cache.Store
}

func NewHub(store db.Store, cache cache.Store) *Hub {
	return &Hub{
		clients:    make(map[uuid.UUID]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		handlers:   make(map[string]MessageHandler),
		store:      store,
		cache:      cache,
	}
}

// RegisterHandler allows external packages (domains) to hook into client actions
// without the gateway knowing anything about business rules.
func (h *Hub) RegisterHandler(msgType string, handler MessageHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.handlers[msgType] = handler
}

func (h *Hub) Run(ctx context.Context) {
	// Listen to pattern-based user events from Redis Pub/Sub
	go h.listenRedisUserEvents(ctx)

	for {
		select {
		case <-ctx.Done():
			return

		case client := <-h.register:
			h.mu.Lock()
			if oldClient, exists := h.clients[client.UserID]; exists {
				oldClient.Close()
			}
			h.clients[client.UserID] = client
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if current, exists := h.clients[client.UserID]; exists && current == client {
				delete(h.clients, client.UserID)
				client.Close()
			}
			h.mu.Unlock()
		}
	}
}

// listenRedisUserEvents subscribes to individual user event streams using pattern matching.
// Channel pattern: "user:*:events"
func (h *Hub) listenRedisUserEvents(ctx context.Context) {
	pattern := "user:*:events"
	sub, err := h.cache.PSubscribe(ctx, pattern)
	if err != nil {
		slog.Error("Failed to psubscribe to user events", "pattern", pattern, "error", err)
		return
	}
	defer sub.Close()

	ch := sub.Channel()

	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			// Pass a detached/non-cancelable context to prevent context cancellation leaks
			go h.dispatchUserEvent(context.WithoutCancel(ctx), msg.Channel, msg.Payload)
		}
	}
}

// dispatchUserEvent parses incoming Redis messages and routes them to the connected client.
func (h *Hub) dispatchUserEvent(ctx context.Context, channel string, payload string) {
	// Extract user ID from channel format "user:<uuid>:events"
	var targetUserIDStr string
	_, err := fmt.Sscanf(channel, "user:%s:events", &targetUserIDStr)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to parse user ID from channel", "channel", channel, "error", err)
		return
	}

	targetUserID, err := uuid.Parse(targetUserIDStr)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to parse UUID from string", "uuid_str", targetUserIDStr, "error", err)
		return
	}

	// Unmarshal the raw JSON payload produced by RegisterOutboxHandlers
	var rawEvent map[string]interface{}
	if err := json.Unmarshal([]byte(payload), &rawEvent); err != nil {
		slog.ErrorContext(ctx, "Failed to unmarshal raw Redis event payload", "error", err)
		return
	}

	eventType, _ := rawEvent["type"].(string)
	eventData, _ := rawEvent["data"]

	innerData, _ := json.Marshal(eventData)
	outboundPayload, _ := json.Marshal(WSMessage{
		Type: eventType,
		Data: innerData,
	})

	h.mu.RLock()
	client, online := h.clients[targetUserID]
	h.mu.RUnlock()

	if online {
		select {
		case client.Send <- outboundPayload:
		default:
			slog.WarnContext(ctx, "Client send buffer full, dropping connection", "user_id", targetUserID)
			go client.Close()
		}
	}
}
