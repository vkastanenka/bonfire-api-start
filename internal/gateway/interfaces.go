package gateway

import (
	"bonfire-api/internal/redis"
	"context"
)

type pubSub interface {
	PublishBatch(ctx context.Context, messages map[string]any) error
	Subscribe(ctx context.Context, channels ...string) (*redis.Subscription, error)
}
