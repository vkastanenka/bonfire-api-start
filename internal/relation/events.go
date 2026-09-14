package relation

import (
	"bonfire-api/internal/channel"
	"bonfire-api/internal/presence"
	"bonfire-api/internal/user"
	"time"

	"github.com/google/uuid"
)

const (
	EventFriendRequestSent = "relation.friend_request_sent"
	EventFriendAdded       = "relation.friend_added"
	EventFriendDeleted     = "relation.friend_deleted"
)

type EventFriendRequestSentPayload struct {
	ActorID   uuid.UUID    `json:"actor_id"`
	PeerID    uuid.UUID    `json:"peer_id"`
	Actor     user.Summary `json:"actor"`
	CreatedAt time.Time    `json:"created_at"`
}

// func NewFriendRequestSentOutboxHandler(gw outbox.Broadcaster) outbox.Handler {
// 	return func(ctx context.Context, payload json.RawMessage) error {
// 		p, err := fields.ParseRawJSON[EventFriendRequestSentPayload](payload)
// 		if err != nil {
// 			return err
// 		}
// 		return outbox.NewUserHandler(gw, EventFriendRequestSent, p.ActorID, p.PeerID)(ctx, payload)
// 	}
// }

type EventFriendAddedPayload struct {
	Friend         *user.User        `json:"friend"`
	FriendPresence presence.Presence `json:"friend_presence"`
	Channel        *channel.Channel  `json:"channel"`
	Member         *channel.Member   `json:"member"`
	CreatedAt      time.Time         `json:"created_at"`
}

type EventFriendDeletedPayload struct {
	ActorID uuid.UUID `json:"actor_id"`
	PeerID  uuid.UUID `json:"peer_id"`
}

// func NewFriendDeletedOutboxHandler(gw outbox.Broadcaster) outbox.Handler {
// 	return func(ctx context.Context, payload json.RawMessage) error {
// 		p, err := fields.ParseRawJSON[EventFriendDeletedPayload](payload)
// 		if err != nil {
// 			return err
// 		}
// 		return outbox.NewUserHandler(gw, EventFriendRequestSent, p.ActorID, p.PeerID)(ctx, payload)
// 	}
// }
