package gateway

import (
	"bonfire-api/internal/fields"
	"bonfire-api/internal/redis"
	"context"
	"encoding/json"

	goredis "github.com/redis/go-redis/v9"
)

const (
	gatewayDomainKey = "gateway:"
)

func gatewayEventsKey(id fields.ID) string {
	return gatewayDomainKey + id.String() + ":events"
}

type Publisher struct {
	client goredis.Cmdable
}

func NewPublisher(client goredis.Cmdable) *Publisher {
	return &Publisher{client: client}
}

func (p *Publisher) PublishEvents(ctx context.Context, events map[fields.ID]Event) error {
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

func SubscribeGatewayEvents(ctx context.Context, client *goredis.Client, nodeID fields.ID) (*redis.Subscription, error) {
	return redis.Subscribe(ctx, client, redis.ScopeGateway, gatewayEventsKey(nodeID))
}
