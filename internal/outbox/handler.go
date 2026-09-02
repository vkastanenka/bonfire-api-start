package outbox

import (
	"context"
	"encoding/json"
	"fmt"

	"bonfire-api/internal/fields"
)

type Handler func(ctx context.Context, payload json.RawMessage) error

type Broadcaster interface {
	BroadcastToPeers(ctx context.Context, actorID fields.ID, eventType string, payload any) error
	BroadcastToFriends(ctx context.Context, actorID fields.ID, eventType string, payload any) error
	BroadcastToUser(ctx context.Context, actorID, targetUserID fields.ID, eventType string, payload any) error
	BroadcastEvent(ctx context.Context, actorID fields.ID, recipientIDs []fields.ID, eventType string, payload any) error
}

func ParsePayload[T any](eventType string, rawMessage json.RawMessage) (*T, error) {
	var msg T
	if err := json.Unmarshal(rawMessage, &msg); err != nil {
		return nil, fmt.Errorf("%w: failed to parse payload for event %s: %v", ErrFatal, eventType, err)
	}
	return &msg, nil
}

// NewPeersHandler broadcasts the raw JSON payload to all peers of actor_id.
func NewPeersHandler(broadcaster Broadcaster, eventType string, rawActorID string) Handler {
	return func(ctx context.Context, payload json.RawMessage) error {
		actorID, err := fields.ParseIDFromString("actor_id", rawActorID)
		if err != nil {
			return fmt.Errorf("%w: invalid actor_id in %s outbox payload: %v", ErrFatal, eventType, err)
		}

		if err := broadcaster.BroadcastToPeers(ctx, actorID, eventType, payload); err != nil {
			return fmt.Errorf("failed to broadcast %s to peers of %s: %w", eventType, actorID, err)
		}

		return nil
	}
}

// NewFriendsHandler broadcasts the raw JSON payload to all friends of actor_id.
func NewFriendsHandler(broadcaster Broadcaster, eventType string, rawActorID string) Handler {
	return func(ctx context.Context, payload json.RawMessage) error {
		actorID, err := fields.ParseIDFromString("actor_id", rawActorID)
		if err != nil {
			return fmt.Errorf("%w: invalid actor_id in %s outbox payload: %v", ErrFatal, eventType, err)
		}

		if err := broadcaster.BroadcastToFriends(ctx, actorID, eventType, payload); err != nil {
			return fmt.Errorf("failed to broadcast %s to friends of %s: %w", eventType, actorID, err)
		}

		return nil
	}
}

// NewUserHandler broadcasts the raw JSON payload to a specific target user_id.
func NewUserHandler(broadcaster Broadcaster, eventType string, rawActorID string, rawRecipientID string) Handler {
	return func(ctx context.Context, payload json.RawMessage) error {
		actorID, err := fields.ParseIDFromString("actor_id", rawActorID)
		if err != nil {
			return fmt.Errorf("%w: invalid actor_id in %s outbox payload: %v", ErrFatal, eventType, err)
		}

		recipientID, err := fields.ParseIDFromString("recipient_id", rawRecipientID)
		if err != nil {
			return fmt.Errorf("%w: invalid recipient_id in %s outbox payload: %v", ErrFatal, eventType, err)
		}

		if err := broadcaster.BroadcastToUser(ctx, actorID, recipientID, eventType, payload); err != nil {
			return fmt.Errorf("failed to broadcast %s from %s to %s: %w", eventType, actorID, recipientID, err)
		}

		return nil
	}
}
