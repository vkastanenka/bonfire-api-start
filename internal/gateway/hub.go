package gateway

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"bonfire-api/internal/presence"
	"bonfire-api/internal/redis"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

type PresenceService interface {
	HandleHeartbeat(ctx context.Context, userID uuid.UUID, nodeID uuid.UUID, sessionID uuid.UUID, newPresence presence.Presence) error
	RegisterNodeSession(ctx context.Context, userID uuid.UUID, nodeID uuid.UUID, sessionID uuid.UUID, presenceStatus presence.Presence) error
	RemoveBatchNodeUsers(ctx context.Context, nodeID uuid.UUID, userIDs []uuid.UUID) error
	UnregisterNodeSession(ctx context.Context, userID uuid.UUID, nodeID uuid.UUID, sessionID uuid.UUID) error
}

var clientBufferLength = 256

type ClientRegistration struct {
	Client   *Client
	Presence presence.Presence
}

type Event struct {
	UserIDs           []uuid.UUID     `json:"user_ids,omitempty"`
	SessionIDs        []uuid.UUID     `json:"session_ids,omitempty"`
	ExcludeSessionIDs []uuid.UUID     `json:"exclude_session_ids,omitempty"`
	Frame             json.RawMessage `json:"frame"`
}

type Hub struct {
	id uuid.UUID

	sessionIdx map[uuid.UUID]*Client
	userIdx    map[uuid.UUID]map[uuid.UUID]*Client

	register   chan ClientRegistration
	unregister chan *Client

	presence PresenceService
	handlers map[string]MessageHandler

	redisClient *goredis.Client
	sub         *redis.Subscription

	mu    sync.RWMutex
	subMu sync.Mutex
}

func NewHub(redisClient *goredis.Client, presence PresenceService) *Hub {
	return &Hub{
		id:          uuid.New(),
		sessionIdx:  make(map[uuid.UUID]*Client),
		userIdx:     make(map[uuid.UUID]map[uuid.UUID]*Client),
		register:    make(chan ClientRegistration, clientBufferLength),
		unregister:  make(chan *Client, clientBufferLength),
		presence:    presence,
		handlers:    make(map[string]MessageHandler),
		redisClient: redisClient,
	}
}

func (h *Hub) ID() uuid.UUID {
	return h.id
}

func (h *Hub) Register(client *Client, presence presence.Presence) {
	h.register <- ClientRegistration{
		Client:   client,
		Presence: presence,
	}
}

func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
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

func (h *Hub) Run(ctx context.Context) {
	go h.listenEvents(ctx)

	for {
		select {
		case <-ctx.Done():
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			h.shutdown(shutdownCtx)
			return

		case reg := <-h.register:
			h.handleRegister(ctx, reg.Client, reg.Presence)

		case client := <-h.unregister:
			h.handleUnregister(ctx, client)
		}
	}
}

func (h *Hub) handleRegister(ctx context.Context, client *Client, presence presence.Presence) {
	h.registerClient(client)

	h.registerNodeSession(ctx, client.UserID, client.SessionID, presence)

	slog.Info("Client connected to gateway",
		"node_id", h.id,
		"user_id", client.UserID,
		"session_id", client.SessionID,
	)
}

func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if oldClient, exists := h.sessionIdx[client.SessionID]; exists {
		oldClient.Close()
	}

	h.sessionIdx[client.SessionID] = client

	sessions, exists := h.userIdx[client.UserID]
	if !exists {
		sessions = make(map[uuid.UUID]*Client)
		h.userIdx[client.UserID] = sessions
	}
	sessions[client.SessionID] = client
}

func (h *Hub) registerNodeSession(ctx context.Context, userID, sessionID uuid.UUID, presence presence.Presence) {
	reqCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer cancel()

	if err := h.presence.RegisterNodeSession(reqCtx, userID, h.id, sessionID, presence); err != nil {
		slog.ErrorContext(ctx, "failed to track user connection", "error", err)
	}
}

func (h *Hub) handleUnregister(ctx context.Context, client *Client) {
	if removed := h.unregisterClient(client); !removed {
		return
	}

	h.unregisterNodeSession(ctx, client.UserID, client.SessionID)
	client.Close()

	slog.Info("Client disconnected from gateway",
		"node_id", h.id,
		"user_id", client.UserID,
		"session_id", client.SessionID,
	)
}

func (h *Hub) unregisterClient(client *Client) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	current, exists := h.sessionIdx[client.SessionID]
	if !exists || current != client {
		return false
	}

	delete(h.sessionIdx, client.SessionID)

	if sessions, ok := h.userIdx[client.UserID]; ok {
		delete(sessions, client.SessionID)
		if len(sessions) == 0 {
			delete(h.userIdx, client.UserID)
		}
	}

	return true
}

func (h *Hub) unregisterNodeSession(ctx context.Context, userID, sessionID uuid.UUID) {
	reqCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer cancel()

	if err := h.presence.UnregisterNodeSession(reqCtx, userID, h.id, sessionID); err != nil {
		slog.ErrorContext(ctx, "failed to untrack user connection", "error", err)
	}
}

func (h *Hub) shutdown(shutdownCtx context.Context) {
	h.unsubscribe()
	h.cleanupNodes(shutdownCtx)
	h.closeAllClients()
}

func (h *Hub) unsubscribe() {
	h.subMu.Lock()
	defer h.subMu.Unlock()
	if h.sub != nil {
		_ = h.sub.Unsubscribe()
	}
}

func (h *Hub) cleanupNodes(ctx context.Context) {
	h.mu.Lock()
	userIDs := make([]uuid.UUID, 0, len(h.userIdx))
	for rawUserID := range h.userIdx {
		userIDs = append(userIDs, rawUserID)
	}
	h.mu.Unlock()

	if len(userIDs) == 0 {
		return
	}

	if err := h.presence.RemoveBatchNodeUsers(ctx, h.id, userIDs); err != nil {
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

func (h *Hub) listenEvents(ctx context.Context) {
	channelKey := gatewayEventsKey(h.id)

	sub, err := redis.Subscribe(ctx, h.redisClient, channelKey)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to subscribe to Redis node channel",
			"id", h.id,
			"channel", channelKey,
			"error", err,
		)
		return
	}

	h.setSubscription(sub)
	h.readEvents(ctx)
}

func (h *Hub) setSubscription(sub *redis.Subscription) {
	h.subMu.Lock()
	h.sub = sub
	h.subMu.Unlock()

	slog.Info("Subscribed to node event stream",
		"id", h.id,
	)
}

func (h *Hub) readEvents(ctx context.Context) {
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
			h.safelyDispatchEvent(ctx, evt.Payload)
		}
	}
}

func (h *Hub) safelyDispatchEvent(ctx context.Context, payload string) {
	defer func() {
		if r := recover(); r != nil {
			slog.ErrorContext(ctx, "recovered from panic during event dispatch",
				"node_id", h.id,
				"error", r,
			)
		}
	}()
	h.dispatchEvent(ctx, payload)
}

func (h *Hub) dispatchEvent(ctx context.Context, payload string) {
	var event Event
	if err := json.Unmarshal([]byte(payload), &event); err != nil {
		slog.ErrorContext(ctx, "failed to unmarshal node event payload",
			"error", err,
			"payload", payload,
		)
		return
	}

	if len(event.SessionIDs) > 0 {
		h.sendToSessions(event.SessionIDs, event.ExcludeSessionIDs, event.Frame)
	}

	if len(event.UserIDs) > 0 {
		h.sendToUsers(event.UserIDs, event.ExcludeSessionIDs, event.Frame)
	}
}

func (h *Hub) sendToSessions(
	sessionIDs []uuid.UUID,
	excludeSessionIDs []uuid.UUID,
	message []byte,
) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, sessionID := range sessionIDs {
		if isSessionExcluded(sessionID, excludeSessionIDs) {
			continue
		}

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

func (h *Hub) sendToUsers(
	userIDs []uuid.UUID,
	excludeSessionIDs []uuid.UUID,
	message []byte,
) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, userID := range userIDs {
		sessions, exists := h.userIdx[userID]
		if !exists {
			continue
		}

		for sessionID, client := range sessions {
			if isSessionExcluded(sessionID, excludeSessionIDs) {
				continue
			}

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

func isSessionExcluded(sessionID uuid.UUID, excluded []uuid.UUID) bool {
	for _, id := range excluded {
		if id == sessionID {
			return true
		}
	}
	return false
}
