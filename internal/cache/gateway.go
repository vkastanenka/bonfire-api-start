package cache

import (
	"bonfire-api/internal/redis"

	"github.com/google/uuid"
	redisdriver "github.com/redis/go-redis/v9"
)

const (
	gatewayDomainKey = "gateway:"
)

func gatewayEventsKey(id uuid.UUID) string {
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
