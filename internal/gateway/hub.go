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

type ClientRegistration struct {
	Client   *Client
	Presence Presence
}

type Event struct {
	ClientIDs []uuid.UUID     `json:"client_ids"`
	Message   json.RawMessage `json:"message"`
}

type Hub struct {
	id uuid.UUID

	register   chan ClientRegistration
	unregister chan *Client

	clients  map[uuid.UUID]*Client
	handlers map[string]MessageHandler

	service *Service
	sub     *redis.Subscription

	mu    sync.RWMutex
	subMu sync.Mutex
}

func NewHub(service *Service) *Hub {
	return &Hub{
		id:         uuid.New(),
		register:   make(chan ClientRegistration, clientBufferLength),
		unregister: make(chan *Client, clientBufferLength),
		clients:    make(map[uuid.UUID]*Client),
		handlers:   make(map[string]MessageHandler),
		service:    service,
	}
}

func (h *Hub) ID() uuid.UUID {
	return h.id
}

// TODO: ENSURE MAXIMUM NUMBER OF CLIENTS PER SESSION

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
	sub, err := h.service.SubscribeToEvents(ctx, h.id)
	if err != nil {
		slog.ErrorContext(ctx, "failed to subscribe to redis gateway events channel",
			"hub_id", h.id,
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
		"hub_id", h.id,
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

	if len(event.ClientIDs) == 0 || len(event.Message) == 0 {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, clientID := range event.ClientIDs {
		client, exists := h.clients[clientID]
		if !exists {
			continue
		}

		select {
		case client.Send <- event.Message:
		default:
			go client.Close()
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

func (h *Hub) clearCache(ctx context.Context) {
	h.mu.Lock()
	if len(h.clients) == 0 {
		h.mu.Unlock()
		return
	}

	clientRefs := make([]ClientRef, 0, len(h.clients))
	for _, client := range h.clients {
		clientRefs = append(clientRefs, ClientRef{
			UserID:    client.UserID,
			SessionID: client.SessionID,
			ClientID:  client.ID,
		})
	}
	h.mu.Unlock()

	if len(clientRefs) == 0 {
		return
	}

	reqCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer cancel()

	if err := h.service.RemoveBatchClients(reqCtx, h.id, clientRefs); err != nil {
		slog.ErrorContext(ctx, "failed to cleanup hub clients in redis", "hub_id", h.id, "error", err)
	}
}

func (h *Hub) closeAllClients() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, client := range h.clients {
		client.Close()
	}

	h.clients = make(map[uuid.UUID]*Client)
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
		"client_id", client.ID,
	)
}

func (h *Hub) setClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if oldClient, exists := h.clients[client.ID]; exists {
		oldClient.Close()
	}

	h.clients[client.ID] = client
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
		"client_id", client.ID,
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
	return true
}
