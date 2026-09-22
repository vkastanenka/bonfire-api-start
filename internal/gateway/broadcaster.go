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

// BroadcastUserEvent is the core pipeline method accepting a slice of excludeSessionIDs.
func (b *Broadcaster) BroadcastUserEvent(
	ctx context.Context,
	recipientIDs []uuid.UUID,
	excludeSessionIDs []uuid.UUID,
	eventType string,
	payload any,
) error {
	if len(recipientIDs) == 0 {
		return nil
	}

	// 1. Resolve active node topology for target users
	nodeToUsers, err := b.presenceCache.GetBatchNodeUsers(ctx, recipientIDs)
	if err != nil || len(nodeToUsers) == 0 {
		return err
	}

	// 2. Pre-marshal inner payload data
	dataBytes, err := json.Marshal(payload)
	if err != nil {
		return errs.Internal("Failed to marshal event payload.").Wrap(err)
	}

	// 3. Pre-marshal wire-format WebSocket frame once for all recipient nodes
	frameBytes, err := json.Marshal(WSMessage{
		Type: eventType,
		Data: dataBytes,
	})
	if err != nil {
		return errs.Internal("Failed to marshal WS wire frame.").Wrap(err)
	}

	// Filter out uuid.Nil entries from excludeSessionIDs upfront
	var cleanExcludes []uuid.UUID
	if len(excludeSessionIDs) > 0 {
		cleanExcludes = make([]uuid.UUID, 0, len(excludeSessionIDs))
		for _, sid := range excludeSessionIDs {
			if sid != uuid.Nil {
				cleanExcludes = append(cleanExcludes, sid)
			}
		}
	}

	// 4. Construct node-targeted envelopes with exclusion list
	nodeEvents := make(map[uuid.UUID]Event, len(nodeToUsers))
	for nodeID, targetUserIDs := range nodeToUsers {
		nodeEvents[nodeID] = Event{
			UserIDs:           targetUserIDs,
			ExcludeSessionIDs: cleanExcludes,
			Frame:             frameBytes,
		}
	}

	return b.publishToNodes(ctx, nodeEvents)
}

// BroadcastToUser broadcasts an event to a single target user with optional session exclusions.
func (b *Broadcaster) BroadcastToUser(
	ctx context.Context,
	recipientID uuid.UUID,
	eventType string,
	payload any,
	excludeSessionIDs ...uuid.UUID,
) error {
	return b.BroadcastUserEvent(ctx, []uuid.UUID{recipientID}, excludeSessionIDs, eventType, payload)
}

// BroadcastToUsers broadcasts an event to multiple target users with optional session exclusions.
func (b *Broadcaster) BroadcastToUsers(
	ctx context.Context,
	recipientIDs []uuid.UUID,
	eventType string,
	payload any,
	excludeSessionIDs ...uuid.UUID,
) error {
	return b.BroadcastUserEvent(ctx, recipientIDs, excludeSessionIDs, eventType, payload)
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
