package gateway

import (
	"bonfire-api/internal/presence"
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func (h *Hub) listenRedisPresence(ctx context.Context) {
	pubsub := h.cache.Subscribe(ctx, presence.PresenceUpdatedChannel)
	defer pubsub.Close()
	ch := pubsub.Channel()

	for msg := range ch {
		go h.broadcastPresenceEvent(ctx, msg.Payload)
	}
}

func (h *Hub) broadcastPresenceEvent(ctx context.Context, payload string) {
	var event presence.PresenceUpdatedEvent
	if err := json.Unmarshal([]byte(payload), &event); err != nil {
		return
	}

	actorID, _ := uuid.Parse(event.UserID)
	dbUUID := pgtype.UUID{Bytes: actorID, Valid: true}

	friends, err := h.store.RelationshipsListFriendsByUserID(ctx, dbUUID)
	if err != nil {
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
				go client.Close()
			}
		}
	}
}
