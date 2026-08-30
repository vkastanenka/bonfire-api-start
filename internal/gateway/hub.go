package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"bonfire-api/internal/db"
	pkgredis "bonfire-api/internal/redis" // Adjust import path to your redis package location

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// MessageHandler defines the signature for processing incoming client messages.
type MessageHandler func(ctx context.Context, client *Client, data json.RawMessage) error

// NodeEventPayload defines the structure of events routed via node-specific channels.
type NodeEventPayload struct {
	Type       string          `json:"type"`
	TargetUser uuid.UUID       `json:"target_user_id"`
	SessionID  *uuid.UUID      `json:"target_session_id,omitempty"` // Optional targeted delivery
	Data       json.RawMessage `json:"data"`
}

// Hub manages active client connections, routes inbound messages,
// and bridges Pub/Sub events from Redis.
type Hub struct {
	nodeID    uuid.UUID
	mu        sync.RWMutex
	clients   map[uuid.UUID]*Client               // Keyed by SessionID
	userIndex map[uuid.UUID]map[uuid.UUID]*Client // UserID -> Set of SessionIDs

	register   chan *Client
	unregister chan *Client

	handlers map[string]MessageHandler
	store    *db.Store

	redisClient *redis.Client
	sub         *pkgredis.Subscription
}

func NewHub(store *db.Store, rdb *redis.Client) *Hub {
	return &Hub{
		nodeID:      uuid.New(),
		clients:     make(map[uuid.UUID]*Client),
		userIndex:   make(map[uuid.UUID]map[uuid.UUID]*Client),
		register:    make(chan *Client, 64),
		unregister:  make(chan *Client, 64),
		handlers:    make(map[string]MessageHandler),
		store:       store,
		redisClient: rdb,
	}
}

// NodeID returns the unique identifier for this gateway instance.
func (h *Hub) NodeID() uuid.UUID {
	return h.nodeID
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
	// Start background Redis listener targeting this specific node
	if h.redisClient != nil {
		go h.listenRedisNodeEvents(ctx)
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

	slog.Info("Client connected to gateway",
		"node_id", h.nodeID,
		"user_id", client.UserID,
		"session_id", client.SessionID,
	)
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
	slog.Info("Client disconnected from gateway",
		"node_id", h.nodeID,
		"user_id", client.UserID,
		"session_id", client.SessionID,
	)
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

// --- Node-Specific Redis Event Routing ---

func (h *Hub) listenRedisNodeEvents(ctx context.Context) {
	channelName := fmt.Sprintf("gateway:%s:events", h.nodeID.String())

	// Exact subscribe to node channel instead of pattern matching
	sub, err := pkgredis.Subscribe(ctx, h.redisClient, pkgredis.ScopeOutboxEvent, channelName)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to subscribe to Redis node channel",
			"node_id", h.nodeID,
			"channel", channelName,
			"error", err,
		)
		return
	}
	h.sub = sub

	slog.InfoContext(ctx, "Subscribed to node event stream",
		"node_id", h.nodeID,
		"channel", channelName,
	)

	ch := h.sub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case evt, ok := <-ch:
			if !ok {
				slog.WarnContext(ctx, "Redis node subscription channel closed", "node_id", h.nodeID)
				return
			}
			h.dispatchNodeEvent(ctx, evt.Payload)
		}
	}
}

func (h *Hub) dispatchNodeEvent(ctx context.Context, payload string) {
	var event NodeEventPayload
	if err := json.Unmarshal([]byte(payload), &event); err != nil {
		slog.ErrorContext(ctx, "Failed to unmarshal Redis node event payload", "error", err)
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

	if event.SessionID != nil {
		h.SendToSession(*event.SessionID, outboundPayload)
		return
	}

	h.SendToUser(event.TargetUser, outboundPayload)
}

// SendToUser pushes a message to all active local WS connections for a specific user.
func (h *Hub) SendToUser(userID uuid.UUID, message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	sessions, exists := h.userIndex[userID]
	if !exists {
		return
	}

	for _, client := range sessions {
		select {
		case client.Send <- message:
		default:
			slog.Warn("Client send buffer full, dropping message",
				"node_id", h.nodeID,
				"user_id", userID,
				"session_id", client.SessionID,
			)
		}
	}
}

// SendToSession delivers a frame directly to a single connection on this node.
func (h *Hub) SendToSession(sessionID uuid.UUID, message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	client, exists := h.clients[sessionID]
	if !exists {
		return
	}

	select {
	case client.Send <- message:
	default:
		slog.Warn("Client send buffer full, dropping direct message",
			"node_id", h.nodeID,
			"session_id", sessionID,
		)
	}
}
