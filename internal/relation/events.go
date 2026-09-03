package relation

import (
	"bonfire-api/internal/fields"
	"bonfire-api/internal/outbox"
	"bonfire-api/internal/user"
	"context"
	"encoding/json"
)

const (
	EventFriendRequestSent     = "relation.friend_request_sent"
	EventFriendRequestAccepted = "relation.friend_request_accepted"
	EventFriendDeleted         = "relation.friend_deleted"
)

type EventFriendRequestSentPayload struct {
	ActorID   string       `json:"actor_id"`
	PeerID    string       `json:"peer_id"`
	Actor     user.Summary `json:"actor"`
	CreatedAt string       `json:"created_at"`
}

type EventFriendDeletedPayload struct {
	ActorID string `json:"actor_id"`
	PeerID  string `json:"peer_id"`
}

func NewFriendRequestSentOutboxHandler(gw outbox.Broadcaster) outbox.Handler {
	return func(ctx context.Context, payload json.RawMessage) error {
		p, err := fields.ParseRawJSON[EventFriendRequestSentPayload](payload)
		if err != nil {
			return err
		}
		return outbox.NewUserHandler(gw, EventFriendRequestSent, p.ActorID, p.PeerID)(ctx, payload)
	}
}

func NewFriendDeletedOutboxHandler(gw outbox.Broadcaster) outbox.Handler {
	return func(ctx context.Context, payload json.RawMessage) error {
		p, err := fields.ParseRawJSON[EventFriendDeletedPayload](payload)
		if err != nil {
			return err
		}
		return outbox.NewUserHandler(gw, EventFriendRequestSent, p.ActorID, p.PeerID)(ctx, payload)
	}
}
