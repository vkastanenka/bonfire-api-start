package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"bonfire-api/internal/redis"

	redisdriver "github.com/redis/go-redis/v9"
)

const (
	defaultMaxBatchSize  = 500
	defaultSingleTimeout = 2 * time.Second
	defaultBatchTimeout  = 3 * time.Second
)

// KeyFunc maps a key type K into its Redis string representation.
type KeyFunc[K comparable] func(key K) string

// JSONCache manages type-safe JSON caching in Redis with automatic chunking and timeouts.
type JSONCache[K comparable, T any] struct {
	client       redisdriver.Cmdable
	scope        redis.Scope
	keyFn        KeyFunc[K]
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
		maxBatchSize: defaultMaxBatchSize,
	}
}

// Get fetches a single item by key.
func (c *JSONCache[K, T]) Get(ctx context.Context, key K) (*T, error) {
	redisKey := c.keyFn(key)

	opCtx, cancel := context.WithTimeout(ctx, defaultSingleTimeout)
	defer cancel()

	bytes, err := c.client.Get(opCtx, redisKey).Bytes()
	if errors.Is(err, redisdriver.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, redis.NewError(err, c.scope)
	}

	var item T
	if err := json.Unmarshal(bytes, &item); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached %s: %w", c.scope, err)
	}

	return &item, nil
}

// GetBatch fetches multiple items using chunked MGet operations.
func (c *JSONCache[K, T]) GetBatch(
	ctx context.Context,
	keys []K,
) (map[K]*T, []K, error) {
	if len(keys) == 0 {
		return map[K]*T{}, nil, nil
	}

	uniqueKeys := deduplicateKeys(keys)
	found := make(map[K]*T, len(uniqueKeys))
	var missing []K

	for i := 0; i < len(uniqueKeys); i += c.maxBatchSize {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}

		end := i + c.maxBatchSize
		if end > len(uniqueKeys) {
			end = len(uniqueKeys)
		}
		chunk := uniqueKeys[i:end]

		err := func() error {
			pipeCtx, cancel := context.WithTimeout(ctx, defaultBatchTimeout)
			defer cancel()

			redisKeys := make([]string, len(chunk))
			for j, k := range chunk {
				redisKeys[j] = c.keyFn(k)
			}

			vals, err := c.client.MGet(pipeCtx, redisKeys...).Result()
			if err != nil && !errors.Is(err, redisdriver.Nil) {
				return redis.NewError(err, c.scope)
			}

			for j, val := range vals {
				key := chunk[j]

				if val == nil {
					missing = append(missing, key)
					continue
				}

				var rawBytes []byte
				switch v := val.(type) {
				case string:
					rawBytes = []byte(v)
				case []byte:
					rawBytes = v
				default:
					missing = append(missing, key)
					continue
				}

				item := new(T)
				if err := json.Unmarshal(rawBytes, item); err != nil {
					// Fall back gracefully to DB if unmarshaling fails for a key
					missing = append(missing, key)
					continue
				}

				found[key] = item
			}
			return nil
		}()

		if err != nil {
			return nil, nil, err
		}
	}

	return found, missing, nil
}

// Set stores a single item by key.
func (c *JSONCache[K, T]) Set(ctx context.Context, key K, item T, ttl time.Duration) error {
	redisKey := c.keyFn(key)

	bytes, err := json.Marshal(item)
	if err != nil {
		return fmt.Errorf("failed to marshal %s: %w", c.scope, err)
	}

	opCtx, cancel := context.WithTimeout(ctx, defaultSingleTimeout)
	defer cancel()

	if err := c.client.Set(opCtx, redisKey, bytes, ttl).Err(); err != nil {
		return redis.NewError(err, c.scope)
	}

	return nil
}

// SetBatch stores items in chunked Redis pipeline transactions.
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

		end := i + c.maxBatchSize
		if end > len(keys) {
			end = len(keys)
		}
		chunk := keys[i:end]

		err := func() error {
			opCtx, cancel := context.WithTimeout(ctx, defaultBatchTimeout)
			defer cancel()

			pipe := c.client.Pipeline()
			for _, k := range chunk {
				bytes, err := json.Marshal(items[k])
				if err != nil {
					return fmt.Errorf("failed to marshal %s: %w", c.scope, err)
				}
				pipe.Set(opCtx, c.keyFn(k), bytes, ttl)
			}

			_, err := pipe.Exec(opCtx)
			if err != nil {
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

// Invalidate removes a single key from cache.
func (c *JSONCache[K, T]) Invalidate(ctx context.Context, key K) error {
	redisKey := c.keyFn(key)

	opCtx, cancel := context.WithTimeout(ctx, defaultSingleTimeout)
	defer cancel()

	if err := c.client.Del(opCtx, redisKey).Err(); err != nil {
		return redis.NewError(err, c.scope)
	}
	return nil
}

// InvalidateBatch bulk-deletes keys using native variadic DEL commands.
func (c *JSONCache[K, T]) InvalidateBatch(ctx context.Context, keys []K) error {
	if len(keys) == 0 {
		return nil
	}

	uniqueKeys := deduplicateKeys(keys)

	for i := 0; i < len(uniqueKeys); i += c.maxBatchSize {
		if err := ctx.Err(); err != nil {
			return err
		}

		end := i + c.maxBatchSize
		if end > len(uniqueKeys) {
			end = len(uniqueKeys)
		}
		chunk := uniqueKeys[i:end]

		err := func() error {
			opCtx, cancel := context.WithTimeout(ctx, defaultBatchTimeout)
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
