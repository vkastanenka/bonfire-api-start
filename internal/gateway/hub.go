package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"bonfire-api/internal/db"
	pkgredis "bonfire-api/internal/redis"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type MessageHandler func(ctx context.Context, client *Client, data json.RawMessage) error

type NodeEventPayload struct {
	Type      string          `json:"type"`
	Data      json.RawMessage `json:"data"`
	UserID    uuid.UUID       `json:"user_id"`
	SessionID *uuid.UUID      `json:"session_id"`
}

type Hub struct {
	nodeID    uuid.UUID
	mu        sync.RWMutex
	clients   map[uuid.UUID]*Client
	userIndex map[uuid.UUID]map[uuid.UUID]*Client

	register   chan *Client
	unregister chan *Client

	handlers map[string]MessageHandler
	store    *db.Store

	redisClient *redis.Client
	subMu       sync.Mutex
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

func (h *Hub) NodeID() uuid.UUID {
	return h.nodeID
}

func (h *Hub) RegisterHandler(msgType string, handler MessageHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.handlers[msgType] = handler
}

func (h *Hub) GetHandler(msgType string) (MessageHandler, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	handler, exists := h.handlers[msgType]
	return handler, exists
}

func (h *Hub) Register(client *Client) {
	h.register <- client
}

func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

func (h *Hub) Run(ctx context.Context) {
	if h.redisClient != nil {
		go h.listenRedisNodeEvents(ctx)
	}

	for {
		select {
		case <-ctx.Done():
			h.shutdown(context.WithoutCancel(ctx))
			return

		case client := <-h.register:
			h.handleRegister(ctx, client)

		case client := <-h.unregister:
			h.handleUnregister(ctx, client)
		}
	}
}

func (h *Hub) handleRegister(ctx context.Context, client *Client) {
	h.mu.Lock()

	if oldClient, exists := h.clients[client.SessionID.UUID()]; exists {
		oldClient.Close()
	}

	h.clients[client.SessionID.UUID()] = client

	sessions, exists := h.userIndex[client.UserID.UUID()]
	isFirstUserSession := !exists || len(sessions) == 0
	if !exists {
		sessions = make(map[uuid.UUID]*Client)
		h.userIndex[client.UserID.UUID()] = sessions
	}
	sessions[client.SessionID.UUID()] = client
	h.mu.Unlock()

	// Update Redis presence state when user first connects to this node
	if isFirstUserSession && h.redisClient != nil {
		presenceKey := fmt.Sprintf("gateway:user_nodes:%s", client.UserID)

		reqCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
		defer cancel()

		if err := h.redisClient.SAdd(reqCtx, presenceKey, h.nodeID.String()).Err(); err != nil {
			slog.ErrorContext(ctx, "failed to track user node presence in redis",
				"user_id", client.UserID,
				"node_id", h.nodeID,
				"error", err,
			)
		}
	}

	slog.Info("Client connected to gateway",
		"node_id", h.nodeID,
		"user_id", client.UserID,
		"session_id", client.SessionID.UUID(),
	)
}

func (h *Hub) handleUnregister(ctx context.Context, client *Client) {
	h.mu.Lock()

	current, exists := h.clients[client.SessionID.UUID()]
	if !exists || current != client {
		h.mu.Unlock()
		return
	}

	delete(h.clients, client.SessionID.UUID())

	isLastUserSession := false
	if sessions, ok := h.userIndex[client.UserID.UUID()]; ok {
		delete(sessions, client.SessionID.UUID())
		if len(sessions) == 0 {
			delete(h.userIndex, client.UserID.UUID())
			isLastUserSession = true
		}
	}
	h.mu.Unlock()

	client.Close()

	// Remove Redis presence state when user has no active sessions left on this node
	if isLastUserSession && h.redisClient != nil {
		presenceKey := fmt.Sprintf("gateway:user_nodes:%s", client.UserID)
		if err := h.redisClient.SRem(ctx, presenceKey, h.nodeID.String()).Err(); err != nil {
			slog.ErrorContext(ctx, "failed to remove user node presence from redis",
				"user_id", client.UserID,
				"node_id", h.nodeID,
				"error", err,
			)
		}
	}

	slog.Info("Client disconnected from gateway",
		"node_id", h.nodeID,
		"user_id", client.UserID,
		"session_id", client.SessionID.UUID(),
	)
}

func (h *Hub) shutdown(shutdownCtx context.Context) {
	h.subMu.Lock()
	if h.sub != nil {
		_ = h.sub.Unsubscribe()
	}
	h.subMu.Unlock()

	h.mu.Lock()
	defer h.mu.Unlock()

	// Clean up user node mappings in Redis for all connected users on this node
	if h.redisClient != nil {
		for userID := range h.userIndex {
			presenceKey := fmt.Sprintf("gateway:user_nodes:%s", userID)
			_ = h.redisClient.SRem(shutdownCtx, presenceKey, h.nodeID.String()).Err()
		}
	}

	for _, client := range h.clients {
		client.Close()
	}
	h.clients = make(map[uuid.UUID]*Client)
	h.userIndex = make(map[uuid.UUID]map[uuid.UUID]*Client)
}

func (h *Hub) listenRedisNodeEvents(ctx context.Context) {
	channelName := fmt.Sprintf("gateway:%s:events", h.nodeID.String())

	sub, err := pkgredis.Subscribe(ctx, h.redisClient, pkgredis.ScopeOutboxEvent, channelName)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to subscribe to Redis node channel",
			"node_id", h.nodeID,
			"channel", channelName,
			"error", err,
		)
		return
	}

	h.subMu.Lock()
	h.sub = sub
	h.subMu.Unlock()

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

	h.SendToUser(event.UserID, outboundPayload)
}

// func (h *Hub) dispatchNodeEvent(ctx context.Context, payload string) {
//     var nodePayload GatewayTargetPayload
//     if err := json.Unmarshal([]byte(payload), &nodePayload); err != nil {
//         slog.ErrorContext(ctx, "Failed to unmarshal node event payload", "error", err)
//         return
//     }

//     msgBytes, err := json.Marshal(nodePayload.Message)
//     if err != nil {
//         return
//     }

//     // Broadcast only to clients connected to THIS gateway instance matching target IDs
//     h.mu.RLock()
//     defer h.mu.RUnlock()

//     for _, userIDStr := range nodePayload.TargetUserIDs {
//         userID := fields.ParseID(userIDStr)
//         if clients, exists := h.userClients[userID]; exists {
//             for client := range clients {
//                 select {
//                 case client.send <- msgBytes:
//                 default:
//                     // Client send buffer full, unregister/drop
//                 }
//             }
//         }
//     }
// }

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
