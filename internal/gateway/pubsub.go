package gateway

import (
	"bonfire-api/internal/redis"
	"context"
	"encoding/json"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

const (
	gatewayDomainKey = "gateway:"
)

func gatewayEventsKey(id uuid.UUID) string {
	return gatewayDomainKey + id.String() + ":events"
}

type Publisher struct {
	client goredis.Cmdable
}

func NewPublisher(client goredis.Cmdable) *Publisher {
	return &Publisher{client: client}
}

func (p *Publisher) PublishEvents(ctx context.Context, events map[uuid.UUID]Event) error {
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

	_, err := p.client.Pipelined(ctx, func(pipe goredis.Pipeliner) error {
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

func SubscribeGatewayEvents(ctx context.Context, client *goredis.Client, nodeID uuid.UUID) (*redis.Subscription, error) {
	return redis.Subscribe(ctx, client, redis.ScopeGateway, gatewayEventsKey(nodeID))
}
