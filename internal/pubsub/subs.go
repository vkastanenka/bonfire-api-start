package pubsub

import (
	"bonfire-api/internal/fields"
	"bonfire-api/internal/redis"
	"context"

	goredis "github.com/redis/go-redis/v9"
)

const (
	gatewayDomainKey = "gateway:"
)

func gatewayEventsKey(id fields.ID) string {
	return gatewayDomainKey + id.String() + ":events"
}

// SubscribeGatewayEvents subscribes to a specific gateway node's event channel and returns a typed channel.
func SubscribeGatewayEvents(ctx context.Context, client *goredis.Client, nodeID fields.ID) (*Subscription, error) {
	sub, err := Subscribe(ctx, client, redis.ScopeGateway, gatewayEventsKey(nodeID))
	if err != nil {
		return nil, err
	}

	return sub, nil
}
