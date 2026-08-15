package cache

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"bonfire-api/internal/errs"
	"bonfire-api/internal/redis"

	redisdriver "github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

const (
	JSONMaxBatchSize  = 500
	JSONSingleTimeout = 500 * time.Millisecond
	JSONBatchTimeout  = 3 * time.Second
)

type KeyFunc[K comparable] func(key K) string

type JSONCache[K comparable, T any] struct {
	client       redisdriver.Cmdable
	scope        redis.Scope
	keyFn        KeyFunc[K]
	sfg          singleflight.Group
	maxBatchSize int
}

func NewJSONCache[K comparable, T any](
	client redisdriver.Cmdable,
	scope redis.Scope,
	keyFn KeyFunc[K],
) *JSONCache[K, T] {
	return &JSONCache[K, T]{
		client:       client,
		scope:        scope,
		keyFn:        keyFn,
		maxBatchSize: JSONMaxBatchSize,
	}
}

func (c *JSONCache[K, T]) Get(ctx context.Context, key K) (*T, error) {
	redisKey := c.keyFn(key)

	val, err, _ := c.sfg.Do(redisKey, func() (any, error) {
		opCtx, cancel := context.WithTimeout(ctx, JSONSingleTimeout)
		defer cancel()

		data, err := c.client.Get(opCtx, redisKey).Bytes()
		if errors.Is(err, redisdriver.Nil) {
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
	})
	if err != nil {
		return nil, err
	}
	if val == nil {
		return nil, nil
	}

	return val.(*T), nil
}

func (c *JSONCache[K, T]) GetBatch(ctx context.Context, keys []K) (map[K]*T, []K, error) {
	if len(keys) == 0 {
		return make(map[K]*T), nil, nil
	}

	found := make(map[K]*T, len(keys))
	var missing []K

	for i := 0; i < len(keys); i += c.maxBatchSize {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}

		end := min(i+c.maxBatchSize, len(keys))
		chunk := keys[i:end]

		redisKeys := make([]string, len(chunk))
		for j, k := range chunk {
			redisKeys[j] = c.keyFn(k)
		}

		err := func() error {
			opCtx, cancel := context.WithTimeout(ctx, JSONBatchTimeout)
			defer cancel()

			vals, err := c.client.MGet(opCtx, redisKeys...).Result()
			if err != nil {
				return redis.NewError(err, c.scope)
			}

			for j, raw := range vals {
				key := chunk[j]

				strVal, ok := raw.(string)
				if !ok || strVal == "" {
					missing = append(missing, key)
					continue
				}

				var item T
				if err := json.Unmarshal([]byte(strVal), &item); err != nil {
					missing = append(missing, key)
					continue
				}

				found[key] = &item
			}
			return nil
		}()

		if err != nil {
			return nil, nil, err
		}
	}

	return found, missing, nil
}

func (c *JSONCache[K, T]) Set(ctx context.Context, key K, item T, ttl time.Duration) error {
	redisKey := c.keyFn(key)

	bytes, err := json.Marshal(item)
	if err != nil {
		return errs.Internal("Failed to marshal cached json.").
			Meta("scope", c.scope.String()).
			Wrap(err)
	}

	opCtx, cancel := context.WithTimeout(ctx, JSONSingleTimeout)
	defer cancel()

	if err := c.client.Set(opCtx, redisKey, bytes, ttl).Err(); err != nil {
		return redis.NewError(err, c.scope)
	}

	return nil
}

func (c *JSONCache[K, T]) SetBatch(ctx context.Context, items map[K]T, ttl time.Duration) error {
	if len(items) == 0 {
		return nil
	}

	keys := make([]K, 0, len(items))
	for k := range items {
		keys = append(keys, k)
	}

	for i := 0; i < len(keys); i += c.maxBatchSize {
		if err := ctx.Err(); err != nil {
			return err
		}

		end := min(i+c.maxBatchSize, len(keys))
		chunk := keys[i:end]

		err := func() error {
			opCtx, cancel := context.WithTimeout(ctx, JSONBatchTimeout)
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

func (c *JSONCache[K, T]) Delete(ctx context.Context, key K) error {
	redisKey := c.keyFn(key)

	opCtx, cancel := context.WithTimeout(ctx, JSONSingleTimeout)
	defer cancel()

	if err := c.client.Del(opCtx, redisKey).Err(); err != nil {
		return redis.NewError(err, c.scope)
	}
	return nil
}

func (c *JSONCache[K, T]) DeleteBatch(ctx context.Context, keys []K) error {
	for i := 0; i < len(keys); i += c.maxBatchSize {
		if err := ctx.Err(); err != nil {
			return err
		}

		end := min(i+c.maxBatchSize, len(keys))
		chunk := keys[i:end]

		err := func() error {
			opCtx, cancel := context.WithTimeout(ctx, JSONBatchTimeout)
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
