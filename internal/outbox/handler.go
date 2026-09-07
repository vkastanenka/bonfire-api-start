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
	BroadcastToSession(ctx context.Context, actorID, targetUserID, targetSessionID fields.ID, eventType string, payload any) error
	BroadcastUserEvent(ctx context.Context, actorID fields.ID, recipientIDs []fields.ID, eventType string, payload any) error
	BroadcastToUsers(ctx context.Context, recipientIDs []fields.ID, excludeSessionIDs []fields.ID, eventType string, payload interface{}) error
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

// NewSessionHandler broadcasts the raw JSON payload to a specific session_id belonging to target_user_id.
func NewSessionHandler(broadcaster Broadcaster, eventType string, rawActorID string, rawTargetUserID string, rawSessionID string) Handler {
	return func(ctx context.Context, payload json.RawMessage) error {
		actorID, err := fields.ParseIDFromString("actor_id", rawActorID)
		if err != nil {
			return fmt.Errorf("%w: invalid actor_id in %s outbox payload: %v", ErrFatal, eventType, err)
		}

		targetUserID, err := fields.ParseIDFromString("target_user_id", rawTargetUserID)
		if err != nil {
			return fmt.Errorf("%w: invalid target_user_id in %s outbox payload: %v", ErrFatal, eventType, err)
		}

		sessionID, err := fields.ParseIDFromString("session_id", rawSessionID)
		if err != nil {
			return fmt.Errorf("%w: invalid session_id in %s outbox payload: %v", ErrFatal, eventType, err)
		}

		if err := broadcaster.BroadcastToSession(ctx, actorID, targetUserID, sessionID, eventType, payload); err != nil {
			return fmt.Errorf("failed to broadcast %s from %s to session %s of user %s: %w", eventType, actorID, sessionID, targetUserID, err)
		}

		return nil
	}
}

// NewUsersHandler broadcasts the raw JSON payload to a list of target recipient_ids,
// with optional excluded session_ids.
func NewUsersHandler(
	broadcaster Broadcaster,
	eventType string,
	rawRecipientIDs []string,
	rawExcludeSessionIDs []string,
) Handler {
	return func(ctx context.Context, payload json.RawMessage) error {
		recipientIDs := make([]fields.ID, 0, len(rawRecipientIDs))
		for idx, rawID := range rawRecipientIDs {
			id, err := fields.ParseIDFromString(fmt.Sprintf("recipient_ids[%d]", idx), rawID)
			if err != nil {
				return fmt.Errorf("%w: invalid recipient_id in %s outbox payload: %v", ErrFatal, eventType, err)
			}
			recipientIDs = append(recipientIDs, id)
		}

		excludeSessionIDs := make([]fields.ID, 0, len(rawExcludeSessionIDs))
		for idx, rawID := range rawExcludeSessionIDs {
			id, err := fields.ParseIDFromString(fmt.Sprintf("exclude_session_ids[%d]", idx), rawID)
			if err != nil {
				return fmt.Errorf("%w: invalid exclude_session_id in %s outbox payload: %v", ErrFatal, eventType, err)
			}
			excludeSessionIDs = append(excludeSessionIDs, id)
		}

		if err := broadcaster.BroadcastToUsers(ctx, recipientIDs, excludeSessionIDs, eventType, payload); err != nil {
			return fmt.Errorf("failed to broadcast %s to target users: %w", eventType, err)
		}

		return nil
	}
}
