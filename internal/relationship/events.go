package relationship

import (
	"context"
	"encoding/json"
	"fmt"

	"bonfire-api/internal/cache"
	"bonfire-api/internal/outbox"

	"github.com/google/uuid"
)

const (
	EventFriendRequestSent     = "relationship.friend_request_sent"
	EventFriendRequestAccepted = "relationship.friend_request_accepted"
	EventFriendRequestDeclined = "relationship.friend_request_declined"
	EventRelationshipRemoved   = "relationship.removed"
	EventUserBlocked           = "relationship.user_blocked"
)

type FriendRequestSentPayload struct {
	ActorID  uuid.UUID `json:"actor_id"`
	TargetID uuid.UUID `json:"target_id"`
}

type FriendRequestAcceptedPayload struct {
	ActorID   uuid.UUID `json:"actor_id"`
	TargetID  uuid.UUID `json:"target_id"`
	ChannelID uuid.UUID `json:"channel_id"`
}

type RelationshipRemovedPayload struct {
	ActorID  uuid.UUID `json:"actor_id"`
	TargetID uuid.UUID `json:"target_id"`
}

type UserBlockedPayload struct {
	ActorID  uuid.UUID `json:"actor_id"`
	TargetID uuid.UUID `json:"target_id"`
}

// RegisterOutboxHandlers wires relationship events to Redis Pub/Sub.
func RegisterOutboxHandlers(w *outbox.Worker, cacheStore *cache.Store) {
	// 1. Friend Request Sent
	w.RegisterHandler(EventFriendRequestSent, func(ctx context.Context, raw json.RawMessage) error {
		var p FriendRequestSentPayload
		if err := json.Unmarshal(raw, &p); err != nil {
			return fmt.Errorf("%w: malformed friend request payload: %v", outbox.ErrFatal, err)
		}

		channel := fmt.Sprintf("user:%s:events", p.TargetID.String())

		wsEvent := map[string]interface{}{
			"type": "FRIEND_REQUEST_RECEIVED",
			"data": map[string]interface{}{
				"from_user_id": p.ActorID,
			},
		}

		return cacheStore.Publish(ctx, channel, wsEvent)
	})

	// 2. Friend Request Accepted
	w.RegisterHandler(EventFriendRequestAccepted, func(ctx context.Context, raw json.RawMessage) error {
		var p FriendRequestAcceptedPayload
		if err := json.Unmarshal(raw, &p); err != nil {
			return fmt.Errorf("%w: malformed request accepted payload: %v", outbox.ErrFatal, err)
		}

		channel := fmt.Sprintf("user:%s:events", p.TargetID.String())

		wsEvent := map[string]interface{}{
			"type": "FRIEND_REQUEST_ACCEPTED",
			"data": map[string]interface{}{
				"user_id": p.ActorID,
			},
		}

		return cacheStore.Publish(ctx, channel, wsEvent)
	})

	// 3. Relationship Removed (Unfriend / Cancel Request)
	w.RegisterHandler(EventRelationshipRemoved, func(ctx context.Context, raw json.RawMessage) error {
		var p RelationshipRemovedPayload
		if err := json.Unmarshal(raw, &p); err != nil {
			return fmt.Errorf("%w: malformed relationship removed payload: %v", outbox.ErrFatal, err)
		}

		// Broadcast to the target user so their friend/request list updates instantly
		channel := fmt.Sprintf("user:%s:events", p.TargetID.String())

		wsEvent := map[string]interface{}{
			"type": "RELATIONSHIP_REMOVED",
			"data": map[string]interface{}{
				"user_id": p.ActorID,
			},
		}

		return cacheStore.Publish(ctx, channel, wsEvent)
	})

	// 4. User Blocked
	w.RegisterHandler(EventUserBlocked, func(ctx context.Context, raw json.RawMessage) error {
		var p UserBlockedPayload
		if err := json.Unmarshal(raw, &p); err != nil {
			return fmt.Errorf("%w: malformed user blocked payload: %v", outbox.ErrFatal, err)
		}

		// Notify the target user they have been blocked (updates UI states / closes connections)
		channel := fmt.Sprintf("user:%s:events", p.TargetID.String())

		wsEvent := map[string]interface{}{
			"type": "USER_BLOCKED",
			"data": map[string]interface{}{
				"user_id": p.ActorID,
			},
		}

		return cacheStore.Publish(ctx, channel, wsEvent)
	})
}
