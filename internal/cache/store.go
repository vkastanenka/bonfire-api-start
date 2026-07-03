package cache

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

func (m *manager) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	bytes, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return m.client.Set(ctx, key, bytes, ttl).Err()
}

func (m *manager) Get(ctx context.Context, key string, dest interface{}) error {
	bytes, err := m.client.Get(ctx, key).Bytes()
	if errors.Is(err, goredis.Nil) { // Fixed: Using explicit goredis alias
		return ErrCacheMiss
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(bytes, dest)
}

func (m *manager) Delete(ctx context.Context, key string) error {
	return m.client.Del(ctx, key).Err()
}

func (m *manager) Exists(ctx context.Context, key string) (bool, error) {
	count, err := m.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (m *manager) Increment(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	pipe := m.client.Pipeline()

	incrCmd := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, ttl)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, err
	}

	return incrCmd.Val(), nil
}
