package relation

import "bonfire-api/internal/user"

const (
	EventFriendRequestSent     = "relation.friend_request_sent"
	EventFriendRequestAccepted = "relation.friend_request_accepted"
	EventRelationDeleted       = "relation.deleted"
	EventUserBlocked           = "relation.user_blocked"
)

type FriendRequestSentPayload struct {
	ActorID   string       `json:"actor_id"`
	PeerID    string       `json:"peer_id"`
	Actor     user.Summary `json:"actor"`
	CreatedAt string       `json:"created_at"`
}

type FriendRequestAcceptedPayload struct {
	ActorID   string `json:"actor_id"`
	PeerID    string `json:"peer_id"`
	ChannelID string `json:"channel_id"`
	UpdatedAt string `json:"updated_at"`
}

type FriendDeletedPayload struct {
	ActorID string `json:"actor_id"`
	PeerID  string `json:"peer_id"`
}

type UserBlockedPayload struct {
	ActorID   string `json:"actor_id"`
	PeerID    string `json:"peer_id"`
	UpdatedAt string `json:"updated_at"`
}

// // RegisterOutboxHandlers wires relation events to Redis Pub/Sub.
// func RegisterOutboxHandlers(w *outbox.Worker, cacheStore *cache.Store) {
// 	// 1. Friend Request Sent
// 	w.RegisterHandler(EventFriendRequestSent, func(ctx context.Context, raw json.RawMessage) error {
// 		var p FriendRequestSentPayload
// 		if err := json.Unmarshal(raw, &p); err != nil {
// 			return fmt.Errorf("%w: malformed friend request payload: %v", outbox.ErrFatal, err)
// 		}

// 		channel := fmt.Sprintf("user:%s:events", p.TargetID.String())

// 		wsEvent := map[string]interface{}{
// 			"type": "FRIEND_REQUEST_RECEIVED",
// 			"data": map[string]interface{}{
// 				"from_user_id": p.ActorID,
// 			},
// 		}

// 		return cacheStore.Publish(ctx, channel, wsEvent)
// 	})

// 	// 2. Friend Request Accepted
// 	w.RegisterHandler(EventFriendRequestAccepted, func(ctx context.Context, raw json.RawMessage) error {
// 		var p FriendRequestAcceptedPayload
// 		if err := json.Unmarshal(raw, &p); err != nil {
// 			return fmt.Errorf("%w: malformed request accepted payload: %v", outbox.ErrFatal, err)
// 		}

// 		channel := fmt.Sprintf("user:%s:events", p.TargetID.String())

// 		wsEvent := map[string]interface{}{
// 			"type": "FRIEND_REQUEST_ACCEPTED",
// 			"data": map[string]interface{}{
// 				"user_id": p.ActorID,
// 			},
// 		}

// 		return cacheStore.Publish(ctx, channel, wsEvent)
// 	})

// 	// 3. relation Removed (Unfriend / Cancel Request)
// 	w.RegisterHandler(EventrelationRemoved, func(ctx context.Context, raw json.RawMessage) error {
// 		var p relationRemovedPayload
// 		if err := json.Unmarshal(raw, &p); err != nil {
// 			return fmt.Errorf("%w: malformed relation removed payload: %v", outbox.ErrFatal, err)
// 		}

// 		// Broadcast to the target user so their friend/request list updates instantly
// 		channel := fmt.Sprintf("user:%s:events", p.TargetID.String())

// 		wsEvent := map[string]interface{}{
// 			"type": "relation_REMOVED",
// 			"data": map[string]interface{}{
// 				"user_id": p.ActorID,
// 			},
// 		}

// 		return cacheStore.Publish(ctx, channel, wsEvent)
// 	})

// 	// 4. User Blocked
// 	w.RegisterHandler(EventUserBlocked, func(ctx context.Context, raw json.RawMessage) error {
// 		var p UserBlockedPayload
// 		if err := json.Unmarshal(raw, &p); err != nil {
// 			return fmt.Errorf("%w: malformed user blocked payload: %v", outbox.ErrFatal, err)
// 		}

// 		// Notify the target user they have been blocked (updates UI states / closes connections)
// 		channel := fmt.Sprintf("user:%s:events", p.TargetID.String())

// 		wsEvent := map[string]interface{}{
// 			"type": "USER_BLOCKED",
// 			"data": map[string]interface{}{
// 				"user_id": p.ActorID,
// 			},
// 		}

// 		return cacheStore.Publish(ctx, channel, wsEvent)
// 	})
// }
