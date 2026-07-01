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
		var event presence.PresenceUpdatedEvent
		if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
			continue
		}

		actorID, err := uuid.Parse(event.UserID)
		if err != nil {
			continue
		}

		// Look up friends who need to know about this status update
		dbUUID := pgtype.UUID{Bytes: actorID, Valid: true}
		friends, err := h.store.RelationshipsListFriendsByUser(ctx, dbUUID)
		if err != nil {
			log.Printf("failed to fetch friends for streaming broadcast: %v", err)
			continue
		}

		// Prepare outbound payload
		outboundPayload, _ := json.Marshal(map[string]interface{}{
			"t": "PRESENCE_UPDATE",
			"d": event,
		})

		// Dispatch payload across active connections
		h.mu.RLock()
		for _, friend := range friends {
			friendID := uuid.UUID(friend.RelatedUserID.Bytes)
			if client, online := h.clients[friendID]; online {
				select {
				case client.Send <- outboundPayload:
				default:
					// Buffer full, unregister failing clients safely
					go func(c *Client) { h.unregister <- c }(client)
				}
			}
		}
		h.mu.RUnlock()
	}
}
