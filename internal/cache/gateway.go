package cache

import (
	"bonfire-api/internal/fields"
	"bonfire-api/internal/redis"

	redisdriver "github.com/redis/go-redis/v9"
)

const (
	gatewayDomainKey = "gateway:"
)

func gatewayEventsKey(id fields.ID) string {
	return gatewayDomainKey + id.String() + ":events"
}

// GatewayCache manages distributed routing state and node presence sets for the gateway cluster.
type GatewayCache struct {
	client redisdriver.Cmdable
	scope  redis.Scope
}

// NewGatewayCache initializes a GatewayCache instance.
func NewGatewayCache(client redisdriver.Cmdable, scope redis.Scope) *GatewayCache {
	return &GatewayCache{
		client: client,
		scope:  scope,
	}
}
