package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// This file defines what your application can do with Redis.
// Your service layers will rely on these interfaces, not the raw Redis client.

// Store handles standard temporary data storage (Pillar 1)
type Store interface {
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Get(ctx context.Context, key string, dest interface{}) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
}

// PresenceTracker handles ephemeral online/offline states (Pillar 2)
type PresenceTracker interface {
	Heartbeat(ctx context.Context, userID string) error
	GetPresence(ctx context.Context, userID string) (bool, error)
	GetBulkPresence(ctx context.Context, userIDs []string) (map[string]bool, error)
}

// MessageBus handles real-time distributed events (Pillar 3)
type MessageBus interface {
	Publish(ctx context.Context, channel string, payload interface{}) error
	// Returns a raw pubsub client for now. As you scale, you can wrap this further.
	Subscribe(ctx context.Context, channel string) *redis.PubSub
}

// Manager is the master interface that implements all Redis capabilities.
type Manager interface {
	Store
	PresenceTracker
	MessageBus
}
