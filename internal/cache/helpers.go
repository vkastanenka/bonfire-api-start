package cache

import (
	"bonfire-api/internal/redis"
	"context"
	"time"

	redisdriver "github.com/redis/go-redis/v9"
)

// get handles the standard Redis Get, cache-miss checks, and error wrapping.
// Returns (data, found, error).
func getKey(ctx context.Context, client redisdriver.Cmdable, key string, scope redis.Scope) ([]byte, bool, error) {
	data, err := client.Get(ctx, key).Bytes()
	if redis.IsCacheMiss(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, redis.NewError(err, scope)
	}
	return data, true, nil
}

func getBatchKeys(ctx context.Context, client redisdriver.Cmdable, keys []string, scope redis.Scope) ([]any, error) {
	vals, err := client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, redis.NewError(err, scope)
	}
	return vals, nil
}

type CacheItem struct {
	Key   string
	Value []byte
}

// setBatchPipeline handles chunked pipelined sets with TTLs and error wrapping.
func setBatchPipeline(ctx context.Context, client redisdriver.Cmdable, items []CacheItem, ttl time.Duration, scope redis.Scope) error {
	if len(items) == 0 {
		return nil
	}

	for i := 0; i < len(items); i += MaxBatchSize {
		if err := ctx.Err(); err != nil {
			return err
		}

		end := min(i+MaxBatchSize, len(items))
		chunk := items[i:end]

		pipe := client.Pipeline()
		for _, item := range chunk {
			pipe.Set(ctx, item.Key, item.Value, ttl)
		}

		if _, err := pipe.Exec(ctx); err != nil {
			return redis.NewError(err, scope)
		}
	}

	return nil
}

// deleteKey handles single-key deletion with scope error wrapping.
func deleteKey(ctx context.Context, client redisdriver.Cmdable, key string, scope redis.Scope) error {
	if err := client.Del(ctx, key).Err(); err != nil {
		return redis.NewError(err, scope)
	}
	return nil
}

// deleteBatchKeys handles chunked batch deletion with context checks and error wrapping.
func deleteBatchKeys(ctx context.Context, client redisdriver.Cmdable, keys []string, scope redis.Scope) error {
	if len(keys) == 0 {
		return nil
	}

	for i := 0; i < len(keys); i += MaxBatchSize {
		if err := ctx.Err(); err != nil {
			return err
		}

		end := min(i+MaxBatchSize, len(keys))
		chunk := keys[i:end]

		if err := client.Del(ctx, chunk...).Err(); err != nil {
			return redis.NewError(err, scope)
		}
	}
	return nil
}

// fetchAndUnmarshal fetches raw bytes from Redis and unmarshals them into a domain model.
// If unmarshaling fails due to corrupted data, it automatically evicts the bad key and returns (nil, nil).
func getAndUnmarshal[T any](
	ctx context.Context,
	client redisdriver.Cmdable,
	key string,
	scope redis.Scope,
	unmarshalFn func([]byte) (*T, error),
) (*T, error) {
	data, found, err := getKey(ctx, client, key, scope)
	if err != nil || !found {
		return nil, err
	}

	entity, err := unmarshalFn(data)
	if err != nil {
		deleteKey(ctx, client, key, scope)
		return nil, nil
	}

	return entity, nil
}

// marshalAndSet marshals a domain entity using the provided marshal function and writes it to Redis.
func marshalAndSet[T any](
	ctx context.Context,
	client redisdriver.Cmdable,
	key string,
	entity *T,
	ttl time.Duration,
	scope redis.Scope,
	marshalFn func(*T) ([]byte, error),
) error {
	bytes, err := marshalFn(entity)
	if err != nil {
		return err
	}

	if err := client.Set(ctx, key, bytes, ttl).Err(); err != nil {
		return redis.NewError(err, scope)
	}

	return nil
}

func toBytes(raw any) ([]byte, bool) {
	if raw == nil {
		return nil, false
	}
	switch v := raw.(type) {
	case string:
		if v == "" {
			return nil, false
		}
		return []byte(v), true
	case []byte:
		if len(v) == 0 {
			return nil, false
		}
		return v, true
	default:
		return nil, false
	}
}
