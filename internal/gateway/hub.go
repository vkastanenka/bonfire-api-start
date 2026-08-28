package gateway

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"sync"

	"bonfire-api/internal/db"
	pkgredis "bonfire-api/internal/redis" // Adjust import path to your redis package location

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// MessageHandler defines the signature for processing incoming client messages.
type MessageHandler func(ctx context.Context, client *Client, data json.RawMessage) error

// Hub manages active client connections, routes inbound messages,
// and bridges Pub/Sub events from Redis.
type Hub struct {
	mu        sync.RWMutex
	clients   map[uuid.UUID]*Client               // Keyed by SessionID
	userIndex map[uuid.UUID]map[uuid.UUID]*Client // UserID -> Set of SessionIDs

	register   chan *Client
	unregister chan *Client

	handlers map[string]MessageHandler
	store    db.Store

	redisClient *redis.Client
	sub         *pkgredis.Subscription
}

func NewHub(store db.Store, rdb *redis.Client) *Hub {
	return &Hub{
		clients:     make(map[uuid.UUID]*Client),
		userIndex:   make(map[uuid.UUID]map[uuid.UUID]*Client),
		register:    make(chan *Client, 64),
		unregister:  make(chan *Client, 64),
		handlers:    make(map[string]MessageHandler),
		store:       store,
		redisClient: rdb,
	}
}

func (h *Hub) RegisterHandler(msgType string, handler MessageHandler) {
	h.handlers[msgType] = handler
}

func (h *Hub) Register(client *Client) {
	h.register <- client
}

func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

func (h *Hub) Run(ctx context.Context) {
	// Start background Redis listener
	if h.redisClient != nil {
		go h.listenRedisUserEvents(ctx)
	}

	for {
		select {
		case <-ctx.Done():
			h.shutdown()
			return

		case client := <-h.register:
			h.handleRegister(client)

		case client := <-h.unregister:
			h.handleUnregister(client)
		}
	}
}

func (h *Hub) handleRegister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Close old session connection if re-registering
	if oldClient, exists := h.clients[client.SessionID]; exists {
		oldClient.Close()
	}

	h.clients[client.SessionID] = client

	sessions, exists := h.userIndex[client.UserID]
	if !exists {
		sessions = make(map[uuid.UUID]*Client)
		h.userIndex[client.UserID] = sessions
	}
	sessions[client.SessionID] = client

	slog.Info("Client connected to gateway", "user_id", client.UserID, "session_id", client.SessionID)
}

func (h *Hub) handleUnregister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	current, exists := h.clients[client.SessionID]
	if !exists || current != client {
		return
	}

	delete(h.clients, client.SessionID)

	if sessions, ok := h.userIndex[client.UserID]; ok {
		delete(sessions, client.SessionID)
		if len(sessions) == 0 {
			delete(h.userIndex, client.UserID)
		}
	}

	client.Close()
	slog.Info("Client disconnected from gateway", "user_id", client.UserID, "session_id", client.SessionID)
}

func (h *Hub) shutdown() {
	if h.sub != nil {
		_ = h.sub.Unsubscribe()
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	for _, client := range h.clients {
		client.Close()
	}
	h.clients = make(map[uuid.UUID]*Client)
	h.userIndex = make(map[uuid.UUID]map[uuid.UUID]*Client)
}

func (h *Hub) DispatchHandler(msgType string) (MessageHandler, bool) {
	handler, exists := h.handlers[msgType]
	return handler, exists
}

// --- Redis Event Routing ---

type RedisUserEvent struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

func (h *Hub) listenRedisUserEvents(ctx context.Context) {
	sub, err := pkgredis.PSubscribe(ctx, h.redisClient, pkgredis.ScopeOutboxEvent, "user:*:events")
	if err != nil {
		slog.ErrorContext(ctx, "Failed to subscribe to Redis user events", "error", err)
		return
	}
	h.sub = sub

	slog.Info("Subscribed to Redis user event stream", "pattern", "user:*:events")

	ch := h.sub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case evt, ok := <-ch:
			if !ok {
				slog.WarnContext(ctx, "Redis subscription channel closed")
				return
			}
			h.dispatchUserEvent(ctx, evt.Channel, evt.Payload)
		}
	}
}

func (h *Hub) dispatchUserEvent(ctx context.Context, channel, payload string) {
	// Pattern format: user:{user_id}:events
	parts := strings.Split(channel, ":")
	if len(parts) < 3 || parts[0] != "user" || parts[2] != "events" {
		slog.ErrorContext(ctx, "Invalid Redis channel pattern received", "channel", channel)
		return
	}

	targetUserID, err := uuid.Parse(parts[1])
	if err != nil {
		slog.ErrorContext(ctx, "Invalid User ID in Redis channel pattern", "channel", channel, "error", err)
		return
	}

	var event RedisUserEvent
	if err := json.Unmarshal([]byte(payload), &event); err != nil {
		slog.ErrorContext(ctx, "Failed to unmarshal Redis event payload", "error", err)
		return
	}

	outboundPayload, err := json.Marshal(WSMessage{
		Type: event.Type,
		Data: event.Data,
	})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to encode outbound WS frame", "error", err)
		return
	}

	h.SendToUser(targetUserID, outboundPayload)
}

// SendToUser pushes a message to all active local WS connections for a specific user.
func (h *Hub) SendToUser(userID uuid.UUID, message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	sessions, exists := h.userIndex[userID]
	if !exists {
		return // User has no active connections on this gateway node
	}

	for _, client := range sessions {
		select {
		case client.Send <- message:
		default:
			// Buffer full: drop frame or close client if connection is stalled
			slog.Warn("Client send buffer full, dropping message", "user_id", userID, "session_id", client.SessionID)
		}
	}
}
