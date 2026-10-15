package gateway

import (
	"bonfire-api/internal/redis"
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
)

var clientBufferLength = 256

func EventsPubSubChannel(id uuid.UUID) string {
	return "gateway:" + id.String() + ":events"
}

type ClientRegistration struct {
	Client   *Client
	Presence Presence
}

type Event struct {
	UserIDs           []uuid.UUID     `json:"user_ids,omitempty"`
	SessionIDs        []uuid.UUID     `json:"session_ids,omitempty"`
	ExcludeSessionIDs []uuid.UUID     `json:"exclude_session_ids,omitempty"`
	Payload           json.RawMessage `json:"payload"`
}

type Hub struct {
	id uuid.UUID

	clients        map[uuid.UUID]*Client
	sessionClients map[uuid.UUID]map[uuid.UUID]*Client
	userClients    map[uuid.UUID]map[uuid.UUID]map[uuid.UUID]*Client

	register   chan ClientRegistration
	unregister chan *Client

	service  *Service
	handlers map[string]MessageHandler

	pubsub pubSub
	sub    *redis.Subscription

	mu    sync.RWMutex
	subMu sync.Mutex
}

func NewHub(pubsub pubSub, service *Service) *Hub {
	return &Hub{
		id:             uuid.New(),
		clients:        make(map[uuid.UUID]*Client),
		sessionClients: make(map[uuid.UUID]map[uuid.UUID]*Client),
		userClients:    make(map[uuid.UUID]map[uuid.UUID]map[uuid.UUID]*Client),
		register:       make(chan ClientRegistration, clientBufferLength),
		unregister:     make(chan *Client, clientBufferLength),
		service:        service,
		handlers:       make(map[string]MessageHandler),
		pubsub:         pubsub,
	}
}

func (h *Hub) ID() uuid.UUID {
	return h.id
}

func (h *Hub) RegisterClient(client *Client, presence Presence) {
	h.register <- ClientRegistration{
		Client:   client,
		Presence: presence,
	}
}

func (h *Hub) UnregisterClient(client *Client) {
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
	go h.subscribe(ctx)

	for {
		select {
		case <-ctx.Done():
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			h.shutdown(shutdownCtx)
			return

		case reg := <-h.register:
			h.registerClient(ctx, reg.Client, reg.Presence)

		case client := <-h.unregister:
			h.unregisterClient(ctx, client)
		}
	}
}

func (h *Hub) subscribe(ctx context.Context) {
	eventsChannel := EventsPubSubChannel(h.id)

	sub, err := h.pubsub.Subscribe(ctx, eventsChannel)
	if err != nil {
		slog.ErrorContext(ctx, "failed to subscribe to redis node channel",
			"id", h.id,
			"channel", eventsChannel,
			"error", err,
		)
		return
	}

	h.setSubscription(sub)
	h.listen(ctx)
}

func (h *Hub) setSubscription(sub *redis.Subscription) {
	h.subMu.Lock()
	h.sub = sub
	h.subMu.Unlock()

	slog.Info("subscribed to node event stream",
		"id", h.id,
	)
}

func (h *Hub) listen(ctx context.Context) {
	h.subMu.Lock()
	sub := h.sub
	h.subMu.Unlock()

	if sub == nil {
		return
	}

	ch := sub.Channel()

	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				slog.WarnContext(ctx, "gateway hub subscription closed", "hub_id", h.id)
				return
			}
			h.safelyWriteEvent(ctx, msg.Payload)
		}
	}
}

func (h *Hub) safelyWriteEvent(ctx context.Context, payload string) {
	defer func() {
		if r := recover(); r != nil {
			slog.ErrorContext(ctx, "recovered from panic during gateway hub event write",
				"hub_id", h.id,
				"error", r,
			)
		}
	}()
	h.writeEvent(ctx, payload)
}

func (h *Hub) writeEvent(ctx context.Context, payload string) {
	var event Event
	if err := json.Unmarshal([]byte(payload), &event); err != nil {
		slog.ErrorContext(ctx, "failed to unmarshal node event payload",
			"error", err,
			"payload", payload,
		)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	if len(event.SessionIDs) > 0 {
		h.writeToSessions(event.SessionIDs, event.Payload)
	} else if len(event.UserIDs) > 0 {
		h.writeToUsers(event.UserIDs, event.ExcludeSessionIDs, event.Payload)
	}
}

func (h *Hub) writeToSessions(sessionIDs []uuid.UUID, payload []byte) {
	for _, sessionID := range sessionIDs {
		sessionClientsMap, exists := h.sessionClients[sessionID]
		if !exists {
			continue
		}

		for _, client := range sessionClientsMap {
			select {
			case client.Send <- payload:
			default:
				go client.Close()
			}
		}
	}
}

func (h *Hub) writeToUsers(userIDs []uuid.UUID, excludeSessionIDs []uuid.UUID, payload []byte) {
	var excludeMap map[uuid.UUID]bool
	if len(excludeSessionIDs) > 0 {
		excludeMap = make(map[uuid.UUID]bool, len(excludeSessionIDs))
		for _, id := range excludeSessionIDs {
			excludeMap[id] = true
		}
	}

	for _, userID := range userIDs {
		userAuthSessions, exists := h.userClients[userID]
		if !exists {
			continue
		}

		for sessionID, sessionClientsMap := range userAuthSessions {
			if excludeMap != nil && excludeMap[sessionID] {
				continue
			}

			for _, client := range sessionClientsMap {
				select {
				case client.Send <- payload:
				default:
					go client.Close()
				}
			}
		}
	}
}

func (h *Hub) shutdown(shutdownCtx context.Context) {
	h.subMu.Lock()
	sub := h.sub
	h.subMu.Unlock()

	if sub != nil {
		if err := sub.Unsubscribe(); err != nil {
			slog.ErrorContext(shutdownCtx, "failed to cleanly unsubscribe hub from redis",
				"hub_id", h.id,
				"error", err,
			)
		}
	}

	h.clearCache(shutdownCtx)
	h.closeAllClients()
}

// TODO
func (h *Hub) clearCache(ctx context.Context) {
	// h.mu.Lock()
	// userIDs := make([]uuid.UUID, 0, len(h.userIdx))
	// for rawUserID := range h.userIdx {
	// 	userIDs = append(userIDs, rawUserID)
	// }
	// h.mu.Unlock()

	// if len(userIDs) == 0 {
	// 	return
	// }

	// if err := h.service.RemoveBatchUsers(ctx, h.id, userIDs); err != nil {
	// 	slog.ErrorContext(ctx, "failed to cleanup redis nodes", "error", err)
	// }
}

func (h *Hub) closeAllClients() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, client := range h.clients {
		client.Close()
	}

	h.clients = make(map[uuid.UUID]*Client)
	h.sessionClients = make(map[uuid.UUID]map[uuid.UUID]*Client)
	h.userClients = make(map[uuid.UUID]map[uuid.UUID]map[uuid.UUID]*Client)
}

func (h *Hub) registerClient(ctx context.Context, client *Client, presence Presence) {
	h.setClient(client)

	reqCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer cancel()

	if err := h.service.RegisterClient(reqCtx, h.id, client.UserID, client.SessionID, client.ID, presence); err != nil {
		slog.ErrorContext(ctx, "failed to track user connection", "error", err)
	}

	slog.Info("client connected to gateway hub",
		"node_id", h.id,
		"user_id", client.UserID,
		"session_id", client.SessionID,
	)
}

func (h *Hub) setClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if oldClient, exists := h.clients[client.ID]; exists {
		oldClient.Close()
	}

	h.clients[client.ID] = client

	sessionClients, exists := h.sessionClients[client.SessionID]
	if !exists {
		sessionClients = make(map[uuid.UUID]*Client)
		h.sessionClients[client.SessionID] = sessionClients
	}
	sessionClients[client.ID] = client

	userClients, exists := h.userClients[client.UserID]
	if !exists {
		userClients = make(map[uuid.UUID]map[uuid.UUID]*Client)
		h.userClients[client.UserID] = userClients
	}

	userClients[client.SessionID] = sessionClients
}

func (h *Hub) unregisterClient(ctx context.Context, client *Client) {
	if removed := h.unsetClient(client); !removed {
		return
	}

	reqCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer cancel()

	if err := h.service.UnregisterClient(reqCtx, h.id, client.UserID, client.SessionID, client.ID); err != nil {
		slog.ErrorContext(ctx, "failed to untrack user connection", "error", err)
	}

	client.Close()

	slog.Info("Client disconnected from gateway",
		"node_id", h.id,
		"user_id", client.UserID,
		"session_id", client.SessionID,
	)
}

func (h *Hub) unsetClient(client *Client) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	current, exists := h.clients[client.ID]
	if !exists || current != client {
		return false
	}

	delete(h.clients, client.ID)

	if sessionClients, ok := h.sessionClients[client.SessionID]; ok {
		delete(sessionClients, client.ID)
		if len(sessionClients) == 0 {
			delete(h.sessionClients, client.SessionID)
		}
	}

	if userSessions, ok := h.userClients[client.UserID]; ok {
		if sessionClientsMap, ok2 := userSessions[client.SessionID]; ok2 {
			delete(sessionClientsMap, client.ID)
			if len(sessionClientsMap) == 0 {
				delete(userSessions, client.SessionID)
			}
		}
		if len(userSessions) == 0 {
			delete(h.userClients, client.UserID)
		}
	}

	return true
}
