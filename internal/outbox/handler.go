package outbox

import (
	"context"
	"encoding/json"
	"fmt"

	"bonfire-api/internal/fields"
)

// PeersBroadcaster broadcasts events to all mutual peers (e.g., presence, profile updates).
type PeersBroadcaster interface {
	BroadcastToPeers(ctx context.Context, actorID fields.ID, eventType string, payload any) error
	BroadcastToFriends(ctx context.Context, actorID fields.ID, eventType string, payload any) error
	BroadcastToUser(ctx context.Context, actorID fields.ID, targetUserID fields.ID, eventType string, payload any) error
}

// FriendsBroadcaster broadcasts events exclusively to confirmed friends.
type FriendsBroadcaster interface {
}

// DirectUserBroadcaster targets a single user (e.g., direct messages, system notifications).
type DirectUserBroadcaster interface {
}

// NewPeersHandler creates a handler that broadcasts to all peers of the actor.
func NewPeersHandler[T any](
	broadcaster PeersBroadcaster,
	eventType string,
	getActorID func(T) string,
) Handler {
	return func(ctx context.Context, payload json.RawMessage) error {
		evt, actorID, err := parsePayloadAndID(payload, eventType, getActorID)
		if err != nil {
			return err
		}

		if err := broadcaster.BroadcastToPeers(ctx, actorID, eventType, evt); err != nil {
			return fmt.Errorf("failed to broadcast %s to peers of user %s: %w", eventType, actorID, err)
		}

		return nil
	}
}

// NewFriendsHandler creates a handler that broadcasts exclusively to friends of the actor.
func NewFriendsHandler[T any](
	broadcaster FriendsBroadcaster,
	eventType string,
	getActorID func(T) string,
) Handler {
	return func(ctx context.Context, payload json.RawMessage) error {
		evt, actorID, err := parsePayloadAndID(payload, eventType, getActorID)
		if err != nil {
			return err
		}

		if err := broadcaster.BroadcastToFriends(ctx, actorID, eventType, evt); err != nil {
			return fmt.Errorf("failed to broadcast %s to friends of user %s: %w", eventType, actorID, err)
		}

		return nil
	}
}

// NewDirectUserHandler creates a handler that targets a specific recipient user ID.
func NewDirectUserHandler[T any](
	broadcaster DirectUserBroadcaster,
	eventType string,
	getActorID func(T) string,
	getRecipientID func(T) string,
) Handler {
	return func(ctx context.Context, payload json.RawMessage) error {
		var evt T
		if err := json.Unmarshal(payload, &evt); err != nil {
			return fmt.Errorf("%w: failed to unmarshal %s outbox event", ErrFatal, eventType)
		}

		actorID, err := fields.ParseIDFromString("actor_id", getActorID(evt))
		if err != nil {
			return fmt.Errorf("%w: invalid actor_id in %s payload: %v", ErrFatal, eventType, err)
		}

		recipientID, err := fields.ParseIDFromString("recipient_id", getRecipientID(evt))
		if err != nil {
			return fmt.Errorf("%w: invalid recipient_id in %s payload: %v", ErrFatal, eventType, err)
		}

		if err := broadcaster.BroadcastToUser(ctx, actorID, recipientID, eventType, evt); err != nil {
			return fmt.Errorf("failed to send direct event %s from %s to %s: %w", eventType, actorID, recipientID, err)
		}

		return nil
	}
}

// Private helper to remove repetitive JSON unmarshaling and actor ID validation.
func parsePayloadAndID[T any](
	payload json.RawMessage,
	eventType string,
	getID func(T) string,
) (T, fields.ID, error) {
	var evt T
	if err := json.Unmarshal(payload, &evt); err != nil {
		return evt, fields.NilID(), fmt.Errorf("%w: failed to unmarshal %s outbox event", ErrFatal, eventType)
	}

	id, err := fields.ParseIDFromString("id", getID(evt))
	if err != nil {
		return evt, fields.NilID(), fmt.Errorf("%w: invalid id in %s payload: %v", ErrFatal, eventType, err)
	}

	return evt, id, nil
}
