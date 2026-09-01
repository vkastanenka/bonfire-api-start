package pubsub

import (
	"context"
	"encoding/json"

	"bonfire-api/internal/fields"
	"bonfire-api/internal/redis"

	goredis "github.com/redis/go-redis/v9"
)

type GatewayPub struct {
	client goredis.Cmdable
}

func NewGatewayPub(client goredis.Cmdable) *GatewayPub {
	return &GatewayPub{client: client}
}

func (p *GatewayPub) PublishNodeEvent(ctx context.Context, nodeID fields.ID, event NodeEvent) error {
	encoded, err := json.Marshal(event)
	if err != nil {
		return redis.NewError(err, redis.ScopeGateway)
	}

	channel := gatewayEventsKey(nodeID)
	if err := p.client.Publish(ctx, channel, encoded).Err(); err != nil {
		return redis.NewError(err, redis.ScopeGateway)
	}

	return nil
}

// PublishNodeEvents broadcasts the exact SAME NodeEvent to multiple nodes in 1 RTT.
func (p *GatewayPub) PublishNodeEvents(ctx context.Context, nodeIDs []fields.ID, event NodeEvent) error {
	if len(nodeIDs) == 0 {
		return nil
	}

	encoded, err := json.Marshal(event)
	if err != nil {
		return redis.NewError(err, redis.ScopeGateway)
	}

	_, err = p.client.Pipelined(ctx, func(pipe goredis.Pipeliner) error {
		for _, id := range nodeIDs {
			pipe.Publish(ctx, gatewayEventsKey(id), encoded)
		}
		return nil
	})
	if err != nil {
		return redis.NewError(err, redis.ScopeGateway)
	}

	return nil
}

// PublishBatchNodeEvents publishes DIFFERENT NodeEvents to their respective node channels in 1 RTT.
func (p *GatewayPub) PublishBatchNodeEvents(ctx context.Context, events map[fields.ID]NodeEvent) error {
	if len(events) == 0 {
		return nil
	}

	// Pre-marshal each node's specific event payload
	marshaledEvents := make(map[fields.ID][]byte, len(events))
	for nodeID, event := range events {
		encoded, err := json.Marshal(event)
		if err != nil {
			return redis.NewError(err, redis.ScopeGateway)
		}
		marshaledEvents[nodeID] = encoded
	}

	// Publish all node-specific payloads concurrently in a single Redis pipeline
	_, err := p.client.Pipelined(ctx, func(pipe goredis.Pipeliner) error {
		for nodeID, encodedPayload := range marshaledEvents {
			pipe.Publish(ctx, gatewayEventsKey(nodeID), encodedPayload)
		}
		return nil
	})
	if err != nil {
		return redis.NewError(err, redis.ScopeGateway)
	}

	return nil
}
