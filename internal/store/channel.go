package store

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type ChannelStore interface {
	Get(ctx context.Context, key string, dest interface{}) error
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key ...string) error
}

type ChannelCache struct {
	store ChannelStore
	ttl   time.Duration
}

func NewChannelCache(store ChannelStore, ttl time.Duration) *ChannelCache {
	return &ChannelCache{store: store, ttl: ttl}
}

func channelKey(id uuid.UUID) string {
	return fmt.Sprintf("channel:{%s}", id.String())
}
