package cache

import (
	"context"
	"time"
	"unsafe"

	"bonfire-api/internal/fields"
	"bonfire-api/internal/redis"

	"github.com/google/uuid"
	redisdriver "github.com/redis/go-redis/v9"
)

type CacheItem struct {
	Key   string
	Value []byte
}

// getKey handles standard Redis Get, cache-miss checks, and scope error wrapping.
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

// getBatchKeys retrieves raw values for a slice of keys via MGet.
func getBatchKeys(ctx context.Context, client redisdriver.Cmdable, keys []string, scope redis.Scope) ([]any, error) {
	vals, err := client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, redis.NewError(err, scope)
	}
	return vals, nil
}

// deleteBatchKeys handles chunked batch deletion natively without pipeline overhead.
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

// setBatchPipeline handles chunked pipelined sets with explicit loop execution.
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

// getAndUnmarshal fetches raw bytes from Redis and unmarshals them into a domain model.
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
		_ = client.Del(ctx, key).Err() // Self-healing eviction; ignore error on corrupted hit
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

// toBytes converts an MGet interface result into a byte slice without heap allocation for strings.
func toBytes(raw any) ([]byte, bool) {
	if raw == nil {
		return nil, false
	}
	switch v := raw.(type) {
	case string:
		if len(v) == 0 {
			return nil, false
		}
		return unsafe.Slice(unsafe.StringData(v), len(v)), true
	case []byte:
		if len(v) == 0 {
			return nil, false
		}
		return v, true
	default:
		return nil, false
	}
}

// getSetIDs retrieves members from a Redis Set key and parses them into fields.ID slices.
func getSetIDs(ctx context.Context, client redisdriver.Cmdable, key string, scope redis.Scope) ([]fields.ID, error) {
	members, err := client.SMembers(ctx, key).Result()
	if redis.IsCacheMiss(err) {
		return nil, nil
	}
	if err != nil {
		return nil, redis.NewError(err, scope)
	}
	if len(members) == 0 {
		return nil, nil
	}

	ids := make([]fields.ID, 0, len(members))
	for _, m := range members {
		if id, parseErr := uuid.Parse(m); parseErr == nil {
			ids = append(ids, fields.ID(id))
		}
	}

	return ids, nil
}

// setSetIDs replaces a Redis Set key with a new slice of IDs inside an atomic TxPipeline.
func setSetIDs(ctx context.Context, client redisdriver.Cmdable, key string, ids []fields.ID, ttl time.Duration, scope redis.Scope) error {
	_, err := client.TxPipelined(ctx, func(pipe redisdriver.Pipeliner) error {
		pipe.Del(ctx, key)
		if len(ids) > 0 {
			members := make([]interface{}, len(ids))
			for i, id := range ids {
				members[i] = id.String()
			}
			pipe.SAdd(ctx, key, members...)
			pipe.Expire(ctx, key, ttl)
		}
		return nil
	})
	if err != nil {
		return redis.NewError(err, scope)
	}
	return nil
}

// addToSetID adds a single field ID to a Redis Set and conditionally updates its TTL if the key exists.
func addToSetID(ctx context.Context, client redisdriver.Cmdable, key string, targetID fields.ID, ttl time.Duration, scope redis.Scope) error {
	_, err := client.TxPipelined(ctx, func(pipe redisdriver.Pipeliner) error {
		pipe.SAdd(ctx, key, targetID.String())
		pipe.ExpireXX(ctx, key, ttl)
		return nil
	})
	if err != nil {
		return redis.NewError(err, scope)
	}
	return nil
}

// removeFromSetID removes a single field ID from a Redis Set.
func removeFromSetID(ctx context.Context, client redisdriver.Cmdable, key string, targetID fields.ID, scope redis.Scope) error {
	if err := client.SRem(ctx, key, targetID.String()).Err(); err != nil {
		return redis.NewError(err, scope)
	}
	return nil
}
