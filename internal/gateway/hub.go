package gateway

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"bonfire-api/internal/fields"
	"bonfire-api/internal/pkg/ptr"
	"bonfire-api/internal/pubsub"
	"bonfire-api/internal/user"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

var clientBufferLength = 256

type MessageHandler func(ctx context.Context, client *Client, data json.RawMessage) error

type Hub struct {
	id fields.ID

	register   chan ClientRegistration
	unregister chan *Client

	handlers map[string]MessageHandler

	sessionIdx map[uuid.UUID]*Client
	userIdx    map[uuid.UUID]map[uuid.UUID]*Client

	redisClient  *goredis.Client
	sub          *pubsub.Subscription
	userCache    UserCache
	gatewayCache GatewayCache
	userService  UserService

	mu    sync.RWMutex
	subMu sync.Mutex
}

func NewHub(redisClient *goredis.Client, userCache UserCache, gatewayCache GatewayCache) *Hub {
	return &Hub{
		id:           fields.ID(uuid.New()),
		register:     make(chan ClientRegistration, clientBufferLength),
		unregister:   make(chan *Client, clientBufferLength),
		sessionIdx:   make(map[uuid.UUID]*Client),
		userIdx:      make(map[uuid.UUID]map[uuid.UUID]*Client),
		handlers:     make(map[string]MessageHandler),
		redisClient:  redisClient,
		userCache:    userCache,
		gatewayCache: gatewayCache,
	}
}

func (h *Hub) ID() fields.ID {
	return h.id
}

func (h *Hub) Register(client *Client, presence user.Presence) {
	h.register <- ClientRegistration{
		Client:   client,
		Presence: presence,
	}
}
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

func (h *Hub) RegisterMessageHandler(msgType string, handler MessageHandler) {
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

func (h *Hub) Run(ctx context.Context) {
	go h.listenRedisNodeEvents(ctx)

	for {
		select {
		case <-ctx.Done():
			h.shutdown(context.WithoutCancel(ctx))
			return

		case reg := <-h.register:
			h.handleRegister(ctx, reg.Client, reg.Presence)

		case client := <-h.unregister:
			h.handleUnregister(ctx, client)
		}
	}
}

func (h *Hub) handleRegister(ctx context.Context, client *Client, presence user.Presence) {
	isFirstUserSession := h.registerClient(client)

	if isFirstUserSession {
		h.trackUserConnection(ctx, client.UserID, presence)
	}

	slog.Info("Client connected to gateway",
		"node_id", h.id,
		"user_id", client.UserID,
		"session_id", client.SessionID.UUID(),
	)
}

func (h *Hub) registerClient(client *Client) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	if oldClient, exists := h.sessionIdx[client.SessionID.UUID()]; exists {
		oldClient.Close()
	}

	h.sessionIdx[client.SessionID.UUID()] = client

	sessions, exists := h.userIdx[client.UserID.UUID()]
	isFirstUserSession := !exists || len(sessions) == 0
	if !exists {
		sessions = make(map[uuid.UUID]*Client)
		h.userIdx[client.UserID.UUID()] = sessions
	}
	sessions[client.SessionID.UUID()] = client

	return isFirstUserSession
}

func (h *Hub) trackUserConnection(ctx context.Context, userID fields.ID, presence user.Presence) {
	reqCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer cancel()

	if err := h.userService.RegisterWSConnection(reqCtx, userID, h.id, presence); err != nil {
		slog.ErrorContext(ctx, "failed to track user connection", "error", err)
	}
}

func (h *Hub) handleUnregister(ctx context.Context, client *Client) {
	isLastUserSession := h.unregisterClient(client)

	// Clean up Redis state BEFORE terminating client channels to prevent write-to-closed-channel panics
	if isLastUserSession {
		h.untrackUserConnection(ctx, client.UserID)
	}

	client.Close()

	slog.Info("Client disconnected from gateway",
		"node_id", h.id,
		"user_id", client.UserID,
		"session_id", client.SessionID.UUID(),
	)
}

func (h *Hub) unregisterClient(client *Client) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	current, exists := h.sessionIdx[client.SessionID.UUID()]
	if !exists || current != client {
		return false
	}

	delete(h.sessionIdx, client.SessionID.UUID())

	isLastUserSession := false
	if sessions, ok := h.userIdx[client.UserID.UUID()]; ok {
		delete(sessions, client.SessionID.UUID())
		if len(sessions) == 0 {
			delete(h.userIdx, client.UserID.UUID())
			isLastUserSession = true
		}
	}

	return isLastUserSession
}

func (h *Hub) untrackUserConnection(ctx context.Context, userID fields.ID) {
	reqCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer cancel()

	if err := h.userService.UnregisterWSConnection(reqCtx, userID, h.id); err != nil {
		slog.ErrorContext(ctx, "failed to untrack user connection", "error", err)
	}
}

func (h *Hub) shutdown(shutdownCtx context.Context) {
	h.unsubscribe()
	h.cleanupRedisNodes(shutdownCtx)
	h.closeAllClients()
}

func (h *Hub) unsubscribe() {
	h.subMu.Lock()
	defer h.subMu.Unlock()
	if h.sub != nil {
		_ = h.sub.Unsubscribe()
	}
}

func (h *Hub) cleanupRedisNodes(ctx context.Context) {
	h.mu.Lock()
	userIDs := make([]fields.ID, 0, len(h.userIdx))
	for rawUserID := range h.userIdx {
		userIDs = append(userIDs, fields.ID(rawUserID))
	}
	h.mu.Unlock()

	if len(userIDs) == 0 {
		return
	}

	reqCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer cancel()

	if err := h.userService.RemoveBatchNode(reqCtx, userIDs, h.id); err != nil {
		slog.ErrorContext(ctx, "failed to cleanup redis nodes", "error", err)
	}
}

func (h *Hub) closeAllClients() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, client := range h.sessionIdx {
		client.Close()
	}
	h.sessionIdx = make(map[uuid.UUID]*Client)
	h.userIdx = make(map[uuid.UUID]map[uuid.UUID]*Client)
}

func (h *Hub) listenRedisNodeEvents(ctx context.Context) {
	sub, err := pubsub.SubscribeGatewayEvents(ctx, h.redisClient, fields.ID(h.ID()))
	if err != nil {
		slog.ErrorContext(ctx, "Failed to subscribe to Redis node channel",
			"id", h.id,
			"error", err,
		)
		return
	}

	h.setSubscription(sub)
	h.readNodeEvents(ctx)
}

func (h *Hub) setSubscription(sub *pubsub.Subscription) {
	h.subMu.Lock()
	h.sub = sub
	h.subMu.Unlock()

	slog.Info("Subscribed to node event stream",
		"id", h.id,
	)
}

func (h *Hub) readNodeEvents(ctx context.Context) {
	ch := h.sub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case evt, ok := <-ch:
			if !ok {
				slog.WarnContext(ctx, "Redis node subscription channel closed", "id", h.id)
				return
			}
			h.dispatchNodeEvent(ctx, evt.Payload)
		}
	}
}

func (h *Hub) dispatchNodeEvent(ctx context.Context, payload string) {
	var event pubsub.NodeEvent
	if err := json.Unmarshal([]byte(payload), &event); err != nil {
		slog.ErrorContext(ctx, "failed to unmarshal node event payload",
			"error", err,
			"payload", payload,
		)
		return
	}

	outboundPayload, err := json.Marshal(WSMessage{
		Type: event.Type,
		Data: event.Data,
	})
	if err != nil {
		slog.ErrorContext(ctx, "failed to encode outbound WS frame",
			"type", event.Type,
			"error", err,
		)
		return
	}

	if len(event.SessionIDs) > 0 {
		h.SendToSessions(event.SessionIDs, outboundPayload)
		return
	}

	if len(event.UserIDs) > 0 {
		h.SendToUsers(event.UserIDs, outboundPayload)
		return
	}

	if event.SessionID != nil {
		h.SendToSession(ptr.From(event.SessionID), outboundPayload)
		return
	}

	if event.UserID != nil {
		h.SendToUser(ptr.From(event.UserID), outboundPayload)
	}
}

func (h *Hub) SendToUser(userID uuid.UUID, message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	sessions, exists := h.userIdx[userID]
	if !exists {
		return
	}

	for _, client := range sessions {
		select {
		case client.Send <- message:
		default:
			slog.Warn("Client send buffer full, dropping message",
				"node_id", h.id,
				"user_id", userID,
				"session_id", client.SessionID,
			)
		}
	}
}

func (h *Hub) SendToUsers(userIDs []uuid.UUID, message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, userID := range userIDs {
		sessions, exists := h.userIdx[userID]
		if !exists {
			continue
		}

		for _, client := range sessions {
			select {
			case client.Send <- message:
			default:
				slog.Warn("Client send buffer full, dropping message",
					"node_id", h.id,
					"user_id", userID,
					"session_id", client.SessionID,
				)
			}
		}
	}
}

func (h *Hub) SendToSession(sessionID uuid.UUID, message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	client, exists := h.sessionIdx[sessionID]
	if !exists {
		return
	}

	select {
	case client.Send <- message:
	default:
		slog.Warn("Client send buffer full, dropping direct message",
			"node_id", h.id,
			"session_id", sessionID,
		)
	}
}

func (h *Hub) SendToSessions(sessionIDs []uuid.UUID, message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, sessionID := range sessionIDs {
		client, exists := h.sessionIdx[sessionID]
		if !exists {
			continue
		}

		select {
		case client.Send <- message:
		default:
			slog.Warn("Client send buffer full, dropping direct message",
				"node_id", h.id,
				"session_id", sessionID,
			)
		}
	}
}
