package relation

import (
	"bonfire-api/internal/channel"
	"bonfire-api/internal/outbox"
	"bonfire-api/internal/presence"
	"bonfire-api/internal/user"
	"context"
	"time"

	"github.com/google/uuid"
)

type Worker interface {
	RegisterHandler(eventType string, handler outbox.Handler)
}

type Broadcaster interface {
	BroadcastToUser(ctx context.Context, recipientID uuid.UUID, eventType string, payload any, excludeSessionIDs ...uuid.UUID) error
	BroadcastToUsers(ctx context.Context, recipientIDs []uuid.UUID, eventType string, payload any, excludeSessionIDs ...uuid.UUID) error
}

const (
	EventFriendRequestSent    = "relation.friend_request_sent"
	EventFriendRequestDeleted = "relation.friend_request_deleted"
	EventFriendAdded          = "relation.friend_added"
	EventFriendDeleted        = "relation.friend_deleted"
)

func RegisterEvents(w Worker, b Broadcaster) {
	w.RegisterHandler(EventFriendRequestSent, newFriendRequestSentEventHandler(b))
	w.RegisterHandler(EventFriendRequestDeleted, newFriendRequestDeletedEventHandler(b))
	w.RegisterHandler(EventFriendAdded, newFriendAddedEventHandler(b))
	w.RegisterHandler(EventFriendDeleted, newFriendDeletedEventHandler(b))
}

type EventFriendRequestSentPayload struct {
	ActorID   uuid.UUID    `json:"actorId"`
	PeerID    uuid.UUID    `json:"peerId"`
	Actor     user.Summary `json:"actor"`
	CreatedAt time.Time    `json:"createdAt"`
}

func newFriendRequestSentEventHandler(b Broadcaster) outbox.Handler {
	return outbox.BindHandler(func(ctx context.Context, m outbox.Metadata, p EventFriendRequestSentPayload) error {
		return b.BroadcastToUser(ctx, p.PeerID, EventFriendRequestSent, p)
	})
}

type EventFriendRequestDeletedPayload struct {
	ActorID uuid.UUID `json:"actorId"`
	PeerID  uuid.UUID `json:"peerId"`
}

func newFriendRequestDeletedEventHandler(b Broadcaster) outbox.Handler {
	return outbox.BindHandler(func(ctx context.Context, m outbox.Metadata, p EventFriendRequestDeletedPayload) error {
		return b.BroadcastToUser(ctx, p.PeerID, EventFriendRequestDeleted, p)
	})
}

type EventFriendAddedPayload struct {
	Friend         user.View         `json:"friend"`
	FriendPresence presence.Presence `json:"friendPresence"`
	Channel        *channel.Channel  `json:"channel"`
	Member         *channel.Member   `json:"member"`
	CreatedAt      time.Time         `json:"createdAt"`
}

func newFriendAddedEventHandler(b Broadcaster) outbox.Handler {
	return outbox.BindHandler(func(ctx context.Context, m outbox.Metadata, p EventFriendAddedPayload) error {
		return b.BroadcastToUsers(ctx, m.RecipientIDs, EventFriendAdded, p)
	})
}

type EventFriendDeletedPayload struct {
	ActorID uuid.UUID `json:"actorId"`
	PeerID  uuid.UUID `json:"peerId"`
}

func newFriendDeletedEventHandler(b Broadcaster) outbox.Handler {
	return outbox.BindHandler(func(ctx context.Context, m outbox.Metadata, p EventFriendDeletedPayload) error {
		return b.BroadcastToUsers(ctx, m.RecipientIDs, EventFriendDeleted, p)
	})
}
