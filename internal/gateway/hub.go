package gateway

import (
	"bonfire-api/internal/cache"
	"bonfire-api/internal/outbox"
	"bonfire-api/internal/presence.go"
	"bonfire-api/internal/repository"
	"bonfire-api/internal/sanitize"
	"bonfire-api/internal/validator"
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	// Added for pgtype.UUID mapping
)

const PresenceUpdatedChannel = "presence:updated"

type Hub struct {
	clients    map[uuid.UUID]*Client
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex

	store       repository.Store
	cache       cache.Manager
	presenceSvc *presence.Service
}

func NewHub(store repository.Store, cache cache.Manager, presenceSvc *presence.Service) *Hub {
	return &Hub{
		clients:     make(map[uuid.UUID]*Client),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		store:       store,
		cache:       cache,
		presenceSvc: presenceSvc,
	}
}

func (h *Hub) Run(ctx context.Context) {
	go h.listenRedisPresence(ctx)

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

			go func(id uuid.UUID, initialPresence presence.Presence) {
				bgCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()

				_ = h.presenceSvc.Heartbeat(bgCtx, id, initialPresence)
				_ = outbox.EmitPresenceUpdated(bgCtx, h.store, outbox.PresenceUpdatedPayload{
					UserID:   id.String(),
					Presence: initialPresence.String(),
				})
			}(client.UserID, client.Presence)

		case client := <-h.unregister:
			h.mu.Lock()
			if current, exists := h.clients[client.UserID]; exists && current == client {
				delete(h.clients, client.UserID)
				client.Close()
			}
			h.mu.Unlock()

			go func(id uuid.UUID) {
				bgCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()

				_ = outbox.EmitPresenceUpdated(bgCtx, h.store, outbox.PresenceUpdatedPayload{
					UserID:   id.String(),
					Presence: presence.PresenceOffline.String(),
				})
			}(client.UserID)
		}
	}
}

func (h *Hub) listenRedisPresence(ctx context.Context) {
	sub, err := h.cache.Subscribe(ctx, PresenceUpdatedChannel)
	if err != nil {
		slog.Error("Failed to subscribe...", "error", err)
		return
	}

	ch := sub.Channel()

	for {
		select {
		case <-ctx.Done():
			_ = sub.Unsubscribe(ctx)
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}

			go h.broadcastPresenceEvent(context.WithoutCancel(ctx), msg)
		}
	}
}

type PresenceUpdatedEvent struct {
	UserID   uuid.UUID `json:"user_id"`
	Presence string    `json:"status"`
}

func (h *Hub) broadcastPresenceEvent(ctx context.Context, msg string) {
	var event PresenceUpdatedEvent
	if err := json.Unmarshal([]byte(msg), &event); err != nil {
		return
	}

	// dbUUID := pgtype.UUID{Bytes: event.UserID, Valid: true}
	// friends, err := h.store.RelationshipsListFriendsByUserID(ctx, dbUUID)
	// if err != nil {
	// 	return
	// }

	// Create the inner data payload
	// innerData, _ := json.Marshal(map[string]string{
	// 	"user_id": event.UserID.String(),
	// 	"status":  event.Presence,
	// })

	// Wrap it in your standard WSMessage envelope
	// outboundPayload, _ := json.Marshal(WSMessage{
	// 	Type: "PRESENCE_UPDATE",
	// 	Data: innerData,
	// })

	h.mu.RLock()
	defer h.mu.RUnlock()

	// for _, friend := range friends {
	// 	friendID := uuid.UUID(friend.PeerID.Bytes)

	// 	if client, online := h.clients[friendID]; online {
	// 		select {
	// 		case client.Send <- outboundPayload:
	// 		default:
	// 			// If the client's send channel is full, they are a dead connection. Boot them.
	// 			go client.Close()
	// 		}
	// 	}
	// }
}

func (h *Hub) handleMessage(client *Client, msg WSMessage) {
	switch msg.Type {
	case "UPDATE_PRESENCE":
		h.handleUpdatePresence(client, msg.Data)
	}
}

func (h *Hub) handleUpdatePresence(client *Client, rawData json.RawMessage) {
	var data struct {
		Presence string `json:"presence" mod:"text" validate:"required,presence"`
	}

	if err := json.Unmarshal(rawData, &data); err != nil {
		return
	}

	sanitize.Normalize(&data)

	if err := validator.Validate(&data); err != nil {
		return
	}

	presenceEnum := presence.ParsePresence(data.Presence)

	client.SetPresence(presenceEnum)

	go func(id uuid.UUID, p presence.Presence) {
		bgCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		_ = h.presenceSvc.Heartbeat(bgCtx, id, p)
		_ = outbox.EmitPresenceUpdated(bgCtx, h.store, outbox.PresenceUpdatedPayload{
			UserID:   id.String(),
			Presence: p.String(),
		})
	}(client.UserID, presenceEnum)
}
