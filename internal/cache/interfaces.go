package cache

import (
	"context"
	"time"
)

// Store handles standard temporary data storage and atomic structures.
type Store interface {
	// Set serializes the value to JSON and stores it with an explicit TTL.
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error

	// Get retrieves a key and unmarshals the raw bytes into the pointer provided by dest.
	// Returns ErrCacheMiss if the key does not exist.
	Get(ctx context.Context, key string, dest interface{}) error

	// Delete removes one or more keys from the cache.
	Delete(ctx context.Context, key string) error

	// Exists checks for the presence of a cache key without pulling its payload.
	Exists(ctx context.Context, key string) (bool, error)

	// Increment atomically increases a numeric counter. If the key is new,
	// it initializes it to 1 and applies the TTL. Subsequent increments
	// do NOT extend the original TTL (Fixed-Window strategy).
	Increment(ctx context.Context, key string, ttl time.Duration) (int64, error)
}

// PresenceTracker handles real-time ephemeral user states and visibility rules.
type PresenceTracker interface {
	// Heartbeat marks a user as actively connected with an internal default sliding TTL.
	Heartbeat(ctx context.Context, userID string) error

	// SetStatus stores a user's persistent custom status selection (e.g., Busy, DND).
	SetStatus(ctx context.Context, userID string, status Presence) error

	// GetStatus pulls a user's base status choice. Defaults to Online if unassigned.
	GetStatus(ctx context.Context, userID string) (Presence, error)

	// GetActivity evaluates the intersection of an active heartbeat and their visibility selection.
	// Returns StatusOffline if the user is invisible or their heartbeat has expired.
	GetActivity(ctx context.Context, userID string) (Presence, error)

	// GetBulkActivity uses an optimized MGet pipeline to compile real-time states
	// for an array of IDs, avoiding N+1 connection overhead.
	GetBulkActivity(ctx context.Context, userIDs []string) (map[string]Presence, error)
}

// Subscription abstracts a real-time event stream away from underlying driver mechanics.
type Subscription interface {
	// Channel exposes a read-only native Go channel delivering raw message payloads.
	Channel() <-chan string

	// Unsubscribe completely terminates the streaming allocation.
	Unsubscribe(ctx context.Context) error
}

// MessageBus orchestrates real-time pub/sub distribution channels across scale-out instances.
type MessageBus interface {
	// Publish broadcasts an event payload serialized to JSON across a specified channel.
	Publish(ctx context.Context, channel string, payload interface{}) error

	// Subscribe attaches a consumer to an event channel, returning an isolated stream interface.
	Subscribe(ctx context.Context, channel string) (Subscription, error)
}

// Manager unifies all caching, tracking, and messaging capabilities into a single contract.
type Manager interface {
	Store
	PresenceTracker
	MessageBus
}
