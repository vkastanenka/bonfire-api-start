package cache

import (
	"context"
	"time"
)

type Store interface {
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Get(ctx context.Context, key string, dest interface{}) error
	MGet(ctx context.Context, keys ...string) ([]interface{}, error)
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	Increment(ctx context.Context, key string, ttl time.Duration) (int64, error)
}

type Subscription interface {
	Channel() <-chan string
	Unsubscribe(ctx context.Context) error
}

type MessageBus interface {
	Publish(ctx context.Context, channel string, payload interface{}) error
	Subscribe(ctx context.Context, channel string) (Subscription, error)
}

type Client interface {
	Store
	MessageBus
}
