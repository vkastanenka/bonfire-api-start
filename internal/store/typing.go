package store

import (
	"context"
	"fmt"
	"time"

	"bonfire-api/internal/cache"

	"github.com/google/uuid"
)

type TypingStore interface {
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key ...string) error
}

type Typing struct {
	store TypingStore
	ttl   time.Duration // e.g., 5 seconds
}

func NewTyping(store TypingStore, ttl time.Duration) *Typing {
	return &Typing{store: store, ttl: ttl}
}

func typingKey(channelID, userID uuid.UUID) string {
	return fmt.Sprintf("typing:{%s}:%s", channelID.String(), userID.String())
}

func (t *Typing) SetTyping(ctx context.Context, channelID, userID uuid.UUID) error {
	k := typingKey(channelID, userID)
	if err := t.store.Set(ctx, k, true, t.ttl); err != nil {
		return cache.NewError(err, cache.ScopeChannel)
	}
	return nil
}
