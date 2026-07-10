package cache

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Store interface {
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Get(ctx context.Context, key string, dest interface{}) error
	MGet(ctx context.Context, keys ...string) ([]interface{}, error)
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	Increment(ctx context.Context, key string, ttl time.Duration) (int64, error)
}

type PresenceTracker interface {
	Heartbeat(ctx context.Context, userID uuid.UUID, p Presence) error
	GetPresence(ctx context.Context, userID uuid.UUID) (Presence, error)
	GetBulkPresence(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]Presence, error)
}

type Subscription interface {
	Channel() <-chan string
	Unsubscribe(ctx context.Context) error
}

type MessageBus interface {
	Publish(ctx context.Context, channel string, payload interface{}) error
	Subscribe(ctx context.Context, channel string) (Subscription, error)
}

type Manager interface {
	Store
	PresenceTracker
	MessageBus
}
