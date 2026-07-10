package gateway

import (
	"bonfire-api/internal/cache"
	"bonfire-api/internal/outbox"
	"bonfire-api/internal/repository"
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
)

const PresenceUpdatedChannel = "presence:updated"

type PresenceUpdatedEvent struct {
	UserID   uuid.UUID `json:"user_id"`
	Presence string    `json:"status"`
}

type Hub struct {
	clients    map[uuid.UUID]*Client
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex

	store repository.Store
	cache cache.Manager
}

func NewHub(store repository.Store, cache cache.Manager) *Hub {
	return &Hub{
		clients:    make(map[uuid.UUID]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		store:      store,
		cache:      cache,
	}
}

func (h *Hub) Run(ctx context.Context) {
	go h.listenRedisPresence(ctx)

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if oldClient, exists := h.clients[client.UserID]; exists {
				oldClient.Close()
			}
			h.clients[client.UserID] = client
			h.mu.Unlock()

			go func(id uuid.UUID, initialPresence cache.Presence) {
				bgCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()

				_ = h.cache.Heartbeat(bgCtx, id, initialPresence)

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
					Presence: cache.PresenceOffline.String(),
				})
			}(client.UserID)
		}
	}
}

func (h *Hub) listenRedisPresence(ctx context.Context) {
	_, err := h.cache.Subscribe(ctx, PresenceUpdatedChannel)
	if err != nil {
		slog.Error("Failed to switch on node presence event consumer stream", "error", err)
		return
	}

	// ch := pubsub.Channel()
	// for msg := range ch {
	// 	go h.broadcastPresenceEvent(context.WithoutCancel(ctx), msg)
	// }
}

// func (h *Hub) broadcastPresenceEvent(ctx context.Context, rawPayload string) {
// 	var event PresenceUpdatedEvent
// 	if err := json.Unmarshal([]byte(rawPayload), &event); err != nil {
// 		return
// 	}

// 	dbUUID := pgtype.UUID{Bytes: event.UserID, Valid: true}
// 	friends, err := h.store.RelationshipsListFriendsByUserID(ctx, dbUUID)
// 	if err != nil {
// 		return
// 	}

// 	outboundPayload, _ := json.Marshal(map[string]interface{}{
// 		"t": "PRESENCE_UPDATE",
// 		"d": map[string]interface{}{
// 			"user_id": event.UserID.String(),
// 			"status":  event.Presence,
// 		},
// 	})

// 	h.mu.RLock()
// 	defer h.mu.RUnlock()

// 	for _, friend := range friends {
// 		friendID := uuid.UUID(friend.PeerID.Bytes)
// 		if client, online := h.clients[friendID]; online {
// 			select {
// 			case client.Send <- outboundPayload:
// 			default:
// 				go client.Close()
// 			}
// 		}
// 	}
// }
