package cache

import (
	"context"
	"encoding/json"
	"time"

	"bonfire-api/internal/errs"
	"bonfire-api/internal/redis"

	redisdriver "github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

const (
	KeyMaxBatchSize  = 500
	KeySingleTimeout = 500 * time.Millisecond
	KeyBatchTimeout  = 3 * time.Second
)

type KeyFunc[K comparable] func(key K) string

type KeyCache[K comparable, T any] struct {
	client redisdriver.Cmdable
	scope  redis.Scope
	keyFn  KeyFunc[K]
	sfg    singleflight.Group
}

func NewKeyCache[K comparable, T any](
	client redisdriver.Cmdable,
	scope redis.Scope,
	keyFn KeyFunc[K],
) *KeyCache[K, T] {
	return &KeyCache[K, T]{
		client: client,
		scope:  scope,
		keyFn:  keyFn,
	}
}

func (c *KeyCache[K, T]) Get(ctx context.Context, key K) (*T, error) {
	redisKey := c.keyFn(key)

	data, err := c.client.Get(ctx, redisKey).Bytes()
	if redis.IsCacheMiss(err) {
		return nil, nil
	}
	if err != nil {
		return nil, redis.NewError(err, c.scope)
	}

	var item T
	if err := json.Unmarshal(data, &item); err != nil {
		return nil, errs.Internal("Failed to unmarshal cached item.").
			Meta("scope", c.scope.String()).
			Wrap(err)
	}

	return &item, nil
}

func (c *KeyCache[K, T]) GetBatch(ctx context.Context, keys []K) (map[K]*T, []K, error) {
	if len(keys) == 0 {
		return map[K]*T{}, nil, nil
	}

	found := make(map[K]*T, len(keys))
	var missing []K

	for i := 0; i < len(keys); i += KeyMaxBatchSize {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}

		end := min(i+KeyMaxBatchSize, len(keys))
		chunk := keys[i:end]

		redisKeys := make([]string, len(chunk))
		for j, k := range chunk {
			redisKeys[j] = c.keyFn(k)
		}

		vals, err := c.client.MGet(ctx, redisKeys...).Result()
		if err != nil {
			return nil, nil, redis.NewError(err, c.scope)
		}

		for j, raw := range vals {
			key := chunk[j]

			if raw == nil {
				missing = append(missing, key)
				continue
			}

			var data []byte
			switch v := raw.(type) {
			case string:
				if v == "" {
					missing = append(missing, key)
					continue
				}
				data = []byte(v)
			case []byte:
				if len(v) == 0 {
					missing = append(missing, key)
					continue
				}
				data = v
			default:
				missing = append(missing, key)
				continue
			}

			var item T
			if err := json.Unmarshal(data, &item); err != nil {
				return nil, nil, errs.Internal("Failed to unmarshal cached batch item.").
					Meta("scope", c.scope.String()).
					Wrap(err)
			}

			found[key] = &item
		}
	}

	if len(missing) > 0 {
		seen := make(map[K]struct{}, len(missing))
		result := make([]K, 0, len(missing))

		for _, k := range missing {
			if _, inFound := found[k]; inFound {
				continue
			}
			if _, inSeen := seen[k]; !inSeen {
				seen[k] = struct{}{}
				result = append(result, k)
			}
		}
		missing = result
	}

	return found, missing, nil
}

func (c *KeyCache[K, T]) Set(ctx context.Context, key K, item T, ttl time.Duration) error {
	redisKey := c.keyFn(key)

	bytes, err := json.Marshal(item)
	if err != nil {
		return errs.Internal("Failed to marshal cached json.").
			Meta("scope", c.scope.String()).
			Wrap(err)
	}

	opCtx, cancel := context.WithTimeout(ctx, KeySingleTimeout)
	defer cancel()

	if err := c.client.Set(opCtx, redisKey, bytes, ttl).Err(); err != nil {
		return redis.NewError(err, c.scope)
	}

	return nil
}

func (c *KeyCache[K, T]) SetBatch(ctx context.Context, items map[K]T, ttl time.Duration) error {
	if len(items) == 0 {
		return nil
	}

	keys := make([]K, 0, len(items))
	for k := range items {
		keys = append(keys, k)
	}

	for i := 0; i < len(keys); i += KeyMaxBatchSize {
		if err := ctx.Err(); err != nil {
			return err
		}

		end := min(i+KeyMaxBatchSize, len(keys))
		chunk := keys[i:end]

		err := func() error {
			opCtx, cancel := context.WithTimeout(ctx, KeyBatchTimeout)
			defer cancel()

			pipe := c.client.Pipeline()
			for _, k := range chunk {
				bytes, err := json.Marshal(items[k])
				if err != nil {
					return errs.Internal("Failed to marshal cached json.").
						Meta("scope", c.scope.String()).
						Wrap(err)
				}
				pipe.Set(opCtx, c.keyFn(k), bytes, ttl)
			}

			if _, err := pipe.Exec(opCtx); err != nil {
				return redis.NewError(err, c.scope)
			}
			return nil
		}()

		if err != nil {
			return err
		}
	}

	return nil
}

func (c *KeyCache[K, T]) Delete(ctx context.Context, key K) error {
	redisKey := c.keyFn(key)

	opCtx, cancel := context.WithTimeout(ctx, KeySingleTimeout)
	defer cancel()

	if err := c.client.Del(opCtx, redisKey).Err(); err != nil {
		return redis.NewError(err, c.scope)
	}
	return nil
}

func (c *KeyCache[K, T]) DeleteBatch(ctx context.Context, keys []K) error {
	for i := 0; i < len(keys); i += KeyMaxBatchSize {
		if err := ctx.Err(); err != nil {
			return err
		}

		end := min(i+KeyMaxBatchSize, len(keys))
		chunk := keys[i:end]

		err := func() error {
			opCtx, cancel := context.WithTimeout(ctx, KeyBatchTimeout)
			defer cancel()

			redisKeys := make([]string, len(chunk))
			for j, k := range chunk {
				redisKeys[j] = c.keyFn(k)
			}

			if err := c.client.Del(opCtx, redisKeys...).Err(); err != nil {
				return redis.NewError(err, c.scope)
			}
			return nil
		}()

		if err != nil {
			return err
		}
	}

	return nil
}

func deduplicateKeys[K comparable](keys []K) []K {
	unique := make([]K, 0, len(keys))
	seen := make(map[K]struct{}, len(keys))
	for _, k := range keys {
		if _, exists := seen[k]; !exists {
			seen[k] = struct{}{}
			unique = append(unique, k)
		}
	}
	return unique
}
