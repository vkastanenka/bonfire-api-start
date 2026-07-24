package relationship

import (
	"context"
	"encoding/json"
	"fmt"

	"bonfire-api/internal/outbox"

	"github.com/google/uuid"
)

// Events emitted by the relationship domain
const (
	EventFriendRequestSent     = "relationship.friend_request.sent"
	EventFriendRequestAccepted = "relationship.friend_request.accepted"
	EventUserBlocked           = "relationship.user.blocked"
	EventRelationshipRemoved   = "relationship.removed"
)

type FriendRequestPayload struct {
	ActorID  uuid.UUID `json:"actor_id"`
	TargetID uuid.UUID `json:"target_id"`
}

type RelationshipCache interface {
	InvalidateUserRelationships(ctx context.Context, userID uuid.UUID) error
}

type RealtimeNotifier interface {
	NotifyUser(ctx context.Context, userID uuid.UUID, eventType string, payload any) error
}

// RegisterOutboxHandlers registers all outbox event handlers for the relationship domain.
func RegisterOutboxHandlers(
	w *outbox.Worker,
	cache RelationshipCache,
	notifier RealtimeNotifier,
) {
	// 1. Friend Request Sent
	w.RegisterHandler(EventFriendRequestSent, func(ctx context.Context, raw json.RawMessage) error {
		var p FriendRequestPayload
		if err := json.Unmarshal(raw, &p); err != nil {
			return fmt.Errorf("%w: malformed friend request payload: %v", outbox.ErrFatal, err)
		}

		// Invalidate relationship caches for BOTH users
		if err := cache.InvalidateUserRelationships(ctx, p.ActorID); err != nil {
			return fmt.Errorf("failed to invalidate actor relationship cache: %w", err)
		}
		if err := cache.InvalidateUserRelationships(ctx, p.TargetID); err != nil {
			return fmt.Errorf("failed to invalidate target relationship cache: %w", err)
		}

		// Notify the target user via WebSockets
		if notifier != nil {
			_ = notifier.NotifyUser(ctx, p.TargetID, EventFriendRequestSent, p)
		}

		return nil
	})

	// 2. Friend Request Accepted (or Auto-Accepted)
	w.RegisterHandler(EventFriendRequestAccepted, func(ctx context.Context, raw json.RawMessage) error {
		var p FriendRequestPayload
		if err := json.Unmarshal(raw, &p); err != nil {
			return fmt.Errorf("%w: malformed friend request accepted payload: %v", outbox.ErrFatal, err)
		}

		if err := cache.InvalidateUserRelationships(ctx, p.ActorID); err != nil {
			return fmt.Errorf("failed to invalidate actor cache: %w", err)
		}
		if err := cache.InvalidateUserRelationships(ctx, p.TargetID); err != nil {
			return fmt.Errorf("failed to invalidate target cache: %w", err)
		}

		if notifier != nil {
			_ = notifier.NotifyUser(ctx, p.ActorID, EventFriendRequestAccepted, p)
			_ = notifier.NotifyUser(ctx, p.TargetID, EventFriendRequestAccepted, p)
		}

		return nil
	})
}
