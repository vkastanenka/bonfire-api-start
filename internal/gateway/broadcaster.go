package gateway

import (
	"context"
	"encoding/json"

	"bonfire-api/internal/pkg/errs"
	"bonfire-api/internal/redis"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

const gatewayDomainKey = "gateway:"

func gatewayEventsKey(id uuid.UUID) string {
	return gatewayDomainKey + id.String() + ":events"
}

type Broadcaster struct {
	rdb           goredis.Cmdable
	presenceCache PresenceCache
}

func NewBroadcaster(rdb goredis.Cmdable, presenceCache PresenceCache) *Broadcaster {
	return &Broadcaster{
		rdb:           rdb,
		presenceCache: presenceCache,
	}
}

// BroadcastToUser broadcasts an event to a single target user with optional session exclusions.
func (b *Broadcaster) BroadcastToUser(
	ctx context.Context,
	recipientID uuid.UUID,
	eventType string,
	payload any,
	excludeSessionIDs ...uuid.UUID,
) error {
	return b.dispatch(ctx, []uuid.UUID{recipientID}, nil, excludeSessionIDs, eventType, payload)
}

// BroadcastToUsers broadcasts an event to multiple target users with optional session exclusions.
func (b *Broadcaster) BroadcastToUsers(
	ctx context.Context,
	recipientIDs []uuid.UUID,
	eventType string,
	payload any,
	excludeSessionIDs ...uuid.UUID,
) error {
	return b.dispatch(ctx, recipientIDs, nil, excludeSessionIDs, eventType, payload)
}

// BroadcastToUserSession sends an event to a single specific session belonging to a user.
func (b *Broadcaster) BroadcastToUserSession(
	ctx context.Context,
	userID, sessionID uuid.UUID,
	eventType string,
	payload any,
) error {
	userSessions := map[uuid.UUID][]uuid.UUID{
		userID: {sessionID},
	}
	return b.BroadcastToUserSessions(ctx, userSessions, eventType, payload)
}

// BroadcastToUserSessions sends targeted events to specific sessions grouped by user (map[userID][]sessionIDs).
func (b *Broadcaster) BroadcastToUserSessions(
	ctx context.Context,
	userSessions map[uuid.UUID][]uuid.UUID,
	eventType string,
	payload any,
	excludeSessionIDs ...uuid.UUID,
) error {
	if len(userSessions) == 0 {
		return nil
	}

	userIDs := make([]uuid.UUID, 0, len(userSessions))
	for userID := range userSessions {
		userIDs = append(userIDs, userID)
	}

	return b.dispatch(ctx, userIDs, userSessions, excludeSessionIDs, eventType, payload)
}

// dispatch is the single core method handling topology resolution, framing, and delivery.
func (b *Broadcaster) dispatch(
	ctx context.Context,
	userIDs []uuid.UUID,
	targetSessions map[uuid.UUID][]uuid.UUID,
	excludeSessionIDs []uuid.UUID,
	eventType string,
	payload any,
) error {
	if len(userIDs) == 0 {
		return nil
	}

	// 1. Resolve active node topology for target users
	nodeToUsers, err := b.presenceCache.GetBatchNodeUsers(ctx, userIDs)
	if err != nil || len(nodeToUsers) == 0 {
		return err
	}

	// 2. Pre-marshal payload and WS wire frame once
	dataBytes, err := json.Marshal(payload)
	if err != nil {
		return errs.Internal("Failed to marshal event payload.").Wrap(err)
	}

	frameBytes, err := json.Marshal(WSMessage{
		Type: eventType,
		Data: dataBytes,
	})
	if err != nil {
		return errs.Internal("Failed to marshal WS wire frame.").Wrap(err)
	}

	cleanExcludes := cleanUUIDSlice(excludeSessionIDs)

	// 3. Construct node-targeted envelopes (user-level OR session-level)
	nodeEvents := make(map[uuid.UUID]Event, len(nodeToUsers))
	for nodeID, activeUserIDs := range nodeToUsers {
		if len(targetSessions) > 0 {
			// Session-targeted mode: aggregate targeted sessionIDs on this node
			var nodeSessionIDs []uuid.UUID
			for _, uid := range activeUserIDs {
				if sIDs, ok := targetSessions[uid]; ok {
					nodeSessionIDs = append(nodeSessionIDs, sIDs...)
				}
			}

			if len(nodeSessionIDs) > 0 {
				nodeEvents[nodeID] = Event{
					SessionIDs:        nodeSessionIDs,
					ExcludeSessionIDs: cleanExcludes,
					Frame:             frameBytes,
				}
			}
		} else {
			// User-targeted mode: target all sessions belonging to userIDs on this node
			nodeEvents[nodeID] = Event{
				UserIDs:           activeUserIDs,
				ExcludeSessionIDs: cleanExcludes,
				Frame:             frameBytes,
			}
		}
	}

	return b.publishToNodes(ctx, nodeEvents)
}

func (b *Broadcaster) publishToNodes(ctx context.Context, events map[uuid.UUID]Event) error {
	if len(events) == 0 {
		return nil
	}

	type encodedPublish struct {
		channel string
		payload []byte
	}
	pubItems := make([]encodedPublish, 0, len(events))

	for nodeID, event := range events {
		encoded, err := json.Marshal(event)
		if err != nil {
			return redis.NewError(err, redis.ScopeGateway)
		}
		pubItems = append(pubItems, encodedPublish{
			channel: gatewayEventsKey(nodeID),
			payload: encoded,
		})
	}

	_, err := b.rdb.Pipelined(ctx, func(pipe goredis.Pipeliner) error {
		for _, item := range pubItems {
			pipe.Publish(ctx, item.channel, item.payload)
		}
		return nil
	})
	if err != nil {
		return redis.NewError(err, redis.ScopeGateway)
	}

	return nil
}

func cleanUUIDSlice(ids []uuid.UUID) []uuid.UUID {
	if len(ids) == 0 {
		return nil
	}
	clean := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if id != uuid.Nil {
			clean = append(clean, id)
		}
	}
	if len(clean) == 0 {
		return nil
	}
	return clean
}
