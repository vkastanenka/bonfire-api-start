package gateway

import (
	"context"
	"encoding/json"
	"log"

	"bonfire-api/internal/presence"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func (h *Hub) listenRedisPresence(ctx context.Context) {
	pubsub := h.cache.Subscribe(ctx, presence.PresenceUpdatedChannel)
	defer pubsub.Close()

	ch := pubsub.Channel()

	for msg := range ch {
		// Offload processing immediately to a worker pool or distinct goroutine
		// so a single DB query lag doesn't backpressure the entire Redis subscription stream.
		go h.broadcastPresenceEvent(ctx, msg.Payload)
	}
}

func (h *Hub) broadcastPresenceEvent(ctx context.Context, payload string) {
	var event presence.PresenceUpdatedEvent
	if err := json.Unmarshal([]byte(payload), &event); err != nil {
		return
	}

	actorID, err := uuid.Parse(event.UserID)
	if err != nil {
		return
	}

	// SCALABILITY NOTE: In a massive scale environment, replace this PostgreSQL query
	// with a Redis call: h.cache.GetFriendsList(ctx, actorID)
	dbUUID := pgtype.UUID{Bytes: actorID, Valid: true}
	friends, err := h.store.RelationshipsListFriendsByUserID(ctx, dbUUID)
	if err != nil {
		log.Printf("failed to fetch friends for streaming broadcast: %v", err)
		return
	}

	outboundPayload, _ := json.Marshal(map[string]interface{}{
		"t": "PRESENCE_UPDATE",
		"d": event,
	})

	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, friend := range friends {
		friendID := uuid.UUID(friend.PeerID.Bytes)
		if client, online := h.clients[friendID]; online {
			select {
			case client.Send <- outboundPayload:
			default:
				// Buffer full; avoid calling unregister channel inside a read-lock context synchronously.
				// Instead, safely trigger client closure directly.
				go client.Close()
			}
		}
	}
}
