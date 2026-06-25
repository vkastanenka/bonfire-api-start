package cache

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// Pillar 1: Standard KV
// This abstracts JSON serialization. Note that we bind these methods to the *redisManager struct.

func (m *redisManager) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	bytes, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return m.client.Set(ctx, key, bytes, ttl).Err()
}

func (m *redisManager) Get(ctx context.Context, key string, dest interface{}) error {
	bytes, err := m.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return ErrCacheMiss
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(bytes, dest)
}

func (m *redisManager) Delete(ctx context.Context, key string) error {
	return m.client.Del(ctx, key).Err()
}

func (m *redisManager) Exists(ctx context.Context, key string) (bool, error) {
	count, err := m.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// Increment atomically bumps an integer counter key by 1.
// If the key does not exist, it initializes it at 1 and applies the provided TTL window.
func (m *redisManager) Increment(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	pipe := m.client.Pipeline()

	incrCmd := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, ttl)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, err
	}

	return incrCmd.Val(), nil
}
