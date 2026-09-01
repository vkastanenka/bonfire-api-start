package gateway

import (
	"bonfire-api/internal/fields"
	"bonfire-api/internal/redis"
	"context"
	"encoding/json"

	goredis "github.com/redis/go-redis/v9"
)

type Publisher struct {
	client goredis.Cmdable
}

func NewPublisher(client goredis.Cmdable) *Publisher {
	return &Publisher{client: client}
}

func (p *Publisher) PublishNodeEvent(ctx context.Context, nodeID fields.ID, event NodeEvent) error {
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

func (p *Publisher) PublishBatchNodeEvents(ctx context.Context, events map[fields.ID]NodeEvent) error {
	if len(events) == 0 {
		return nil
	}

	marshaledEvents := make(map[fields.ID][]byte, len(events))
	for nodeID, event := range events {
		encoded, err := json.Marshal(event)
		if err != nil {
			return redis.NewError(err, redis.ScopeGateway)
		}
		marshaledEvents[nodeID] = encoded
	}

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

const (
	gatewayDomainKey = "gateway:"
)

func gatewayEventsKey(id fields.ID) string {
	return gatewayDomainKey + id.String() + ":events"
}

func SubscribeGatewayEvents(ctx context.Context, client *goredis.Client, nodeID fields.ID) (*redis.Subscription, error) {
	sub, err := redis.Subscribe(ctx, client, redis.ScopeGateway, gatewayEventsKey(nodeID))
	if err != nil {
		return nil, err
	}

	return sub, nil
}
