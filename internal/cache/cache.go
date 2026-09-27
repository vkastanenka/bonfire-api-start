package cache

import (
	"context"
	"errors"
	"fmt"
	"time"
	"unsafe"

	"bonfire-api/internal/redis"

	"github.com/google/uuid"
	redisdriver "github.com/redis/go-redis/v9"
)

var maxBatchSize = 500

type CacheItem struct {
	Key   string
	Value []byte
}

// getKey handles standard Redis Get, cache-miss checks, and scope error wrapping.
func getKey(ctx context.Context, scope redis.Scope, client redisdriver.Cmdable, key string) ([]byte, bool, error) {
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
func getBatchKeys(ctx context.Context, scope redis.Scope, client redisdriver.Cmdable, keys []string) ([]any, error) {
	vals, err := client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, redis.NewError(err, scope)
	}
	return vals, nil
}

// deleteBatchKeys handles chunked batch deletion natively without pipeline overhead.
func deleteBatchKeys(ctx context.Context, scope redis.Scope, client redisdriver.Cmdable, keys []string) error {
	for i := 0; i < len(keys); i += maxBatchSize {
		if err := ctx.Err(); err != nil {
			return err
		}

		end := min(i+maxBatchSize, len(keys))
		chunk := keys[i:end]

		if err := client.Del(ctx, chunk...).Err(); err != nil {
			return redis.NewError(err, scope)
		}
	}
	return nil
}

// setBatchPipeline executes chunked pipelined SET commands.
func setBatchPipeline(
	ctx context.Context,
	scope redis.Scope,
	client redisdriver.Cmdable,
	items []CacheItem,
	ttl time.Duration,
) error {
	for i := 0; i < len(items); i += maxBatchSize {
		if err := ctx.Err(); err != nil {
			return err
		}

		end := min(i+maxBatchSize, len(items))
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
func getAndUnmarshal[T any](
	ctx context.Context,
	scope redis.Scope,
	client redisdriver.Cmdable,
	key string,
	unmarshalFn func([]byte) (*T, error),
) (*T, error) {
	data, found, err := getKey(ctx, scope, client, key)
	if err != nil || !found {
		return nil, err
	}

	entity, err := unmarshalFn(data)
	if err != nil {
		_ = client.Del(ctx, key).Err()
		return nil, redis.NewError(fmt.Errorf("%w: %v", redis.ErrCorruptedData, err), scope)
	}

	return entity, nil
}

// marshalAndSet marshals a domain entity using the provided marshal function and writes it to Redis.
func marshalAndSet[T any](
	ctx context.Context,
	scope redis.Scope,
	client redisdriver.Cmdable,
	key string,
	entity *T,
	ttl time.Duration,
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

// getAndUnmarshalBatch generic helper that fetches keys using MGET in chunks.
func getAndUnmarshalBatch[K comparable, T any](
	ctx context.Context,
	scope redis.Scope,
	client redisdriver.Cmdable,
	identifiers []K,
	keyFn func(K) string,
	unmarshalFn func([]byte) (*T, error),
) (map[K]*T, []K, error) {
	found := make(map[K]*T, len(identifiers))
	missing := make([]K, 0, len(identifiers))
	var corrupted []string

	for i := 0; i < len(identifiers); i += maxBatchSize {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}

		end := min(i+maxBatchSize, len(identifiers))
		idChunk := identifiers[i:end]

		keyChunk := make([]string, len(idChunk))
		for k, id := range idChunk {
			keyChunk[k] = keyFn(id)
		}

		vals, err := getBatchKeys(ctx, scope, client, keyChunk)
		if err != nil {
			return nil, nil, err
		}

		for j, raw := range vals {
			id := idChunk[j]
			key := keyChunk[j]

			data, ok := toBytes(raw)
			if !ok || len(data) == 0 {
				missing = append(missing, id)
				continue
			}

			entity, err := unmarshalFn(data)
			if err != nil {
				corrupted = append(corrupted, key)
				missing = append(missing, id)
				continue
			}

			found[id] = entity
		}
	}

	if len(corrupted) > 0 {
		_ = deleteBatchKeys(ctx, scope, client, corrupted)
	}

	return found, missing, nil
}

// marshalAndSetBatch accepts a map of entities, marshals non-nil entries, and writes them to Redis using chunked pipelines.
func marshalAndSetBatch[K comparable, T any](
	ctx context.Context,
	scope redis.Scope,
	client redisdriver.Cmdable,
	entities map[K]*T,
	keyFn func(K) string,
	ttl time.Duration,
	marshalFn func(*T) ([]byte, error),
) error {
	if len(entities) == 0 {
		return nil
	}

	items := make([]CacheItem, 0, len(entities))
	for id, entity := range entities {
		if entity == nil {
			continue
		}

		bytes, err := marshalFn(entity)
		if err != nil {
			return err
		}

		items = append(items, CacheItem{
			Key:   keyFn(id),
			Value: bytes,
		})
	}

	if len(items) == 0 {
		return nil
	}

	return setBatchPipeline(ctx, scope, client, items, ttl)
}

// deleteBatch transforms typed domain identifiers into Redis keys and passes them to deleteBatchKeys.
func deleteBatch[K comparable](
	ctx context.Context,
	scope redis.Scope,
	client redisdriver.Cmdable,
	identifiers []K,
	keyFn func(K) string,
) error {
	keys := make([]string, len(identifiers))
	for i, id := range identifiers {
		keys[i] = keyFn(id)
	}

	return deleteBatchKeys(ctx, scope, client, keys)
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

// getSetIDs retrieves members from a Redis Set key and parses them into uuid.UUID slices.
func getSetIDs(
	ctx context.Context,
	scope redis.Scope,
	client redisdriver.Cmdable,
	key string,
) ([]uuid.UUID, error) {
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

	ids := make([]uuid.UUID, 0, len(members))
	for _, m := range members {
		id, err := uuid.Parse(m)
		if err != nil {
			continue
		}
		ids = append(ids, id)
	}

	return ids, nil
}

// setSetIDs atomically replaces a Redis Set key in a pipeline.
func setSetIDs(
	ctx context.Context,
	scope redis.Scope,
	client redisdriver.Cmdable,
	key string,
	ids []uuid.UUID,
	ttl time.Duration,
) error {
	if len(ids) == 0 {
		if err := client.Del(ctx, key).Err(); err != nil {
			return redis.NewError(err, scope)
		}
		return nil
	}

	members := make([]any, len(ids))
	for i, id := range ids {
		members[i] = id.String()
	}

	_, err := client.TxPipelined(ctx, func(pipe redisdriver.Pipeliner) error {
		pipe.Del(ctx, key)
		pipe.SAdd(ctx, key, members...)
		pipe.Expire(ctx, key, ttl)
		return nil
	})
	if err != nil {
		return redis.NewError(err, scope)
	}

	return nil
}

// addToSetIDs adds one or more UUIDs to a single Redis Set and sets TTL.
func addToSetIDs(
	ctx context.Context,
	scope redis.Scope,
	client redisdriver.Cmdable,
	key string,
	ttl time.Duration,
	ids ...uuid.UUID,
) error {
	if len(ids) == 0 {
		return nil
	}

	members := make([]any, len(ids))
	for i, id := range ids {
		members[i] = id.String()
	}

	_, err := client.TxPipelined(ctx, func(pipe redisdriver.Pipeliner) error {
		pipe.SAdd(ctx, key, members...)
		pipe.Expire(ctx, key, ttl)
		return nil
	})
	if err != nil {
		return redis.NewError(err, scope)
	}

	return nil
}

// addToSetIDsPipelined adds UUIDs to multiple set keys with a shared TTL in a single pipeline execution.
func addToSetIDsPipelined(
	ctx context.Context,
	scope redis.Scope,
	client redisdriver.Cmdable,
	additions map[string][]uuid.UUID,
	ttl time.Duration,
) error {
	if len(additions) == 0 {
		return nil
	}

	pipe := client.Pipeline()
	for key, ids := range additions {
		if len(ids) == 0 {
			continue
		}
		members := make([]any, len(ids))
		for i, id := range ids {
			members[i] = id.String()
		}
		pipe.SAdd(ctx, key, members...)
		pipe.Expire(ctx, key, ttl)
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return redis.NewError(err, scope)
	}

	return nil
}

// removeFromSetIDs removes one or more UUIDs from a single Redis Set.
func removeFromSetIDs(
	ctx context.Context,
	scope redis.Scope,
	client redisdriver.Cmdable,
	key string,
	ids ...uuid.UUID,
) error {
	if len(ids) == 0 {
		return nil
	}

	members := make([]any, len(ids))
	for i, id := range ids {
		members[i] = id.String()
	}

	if err := client.SRem(ctx, key, members...).Err(); err != nil {
		return redis.NewError(err, scope)
	}

	return nil
}

// removeFromSetIDsPipelined handles bulk removals across multiple set keys in one pipeline trip.
func removeFromSetIDsPipelined(
	ctx context.Context,
	scope redis.Scope,
	client redisdriver.Cmdable,
	removals map[string][]uuid.UUID,
) error {
	if len(removals) == 0 {
		return nil
	}

	pipe := client.Pipeline()
	for key, ids := range removals {
		if len(ids) == 0 {
			continue
		}
		members := make([]any, len(ids))
		for i, id := range ids {
			members[i] = id.String()
		}
		pipe.SRem(ctx, key, members...)
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return redis.NewError(err, scope)
	}

	return nil
}

// deleteSet explicitly deletes a set index key.
func deleteSet(
	ctx context.Context,
	scope redis.Scope,
	client redisdriver.Cmdable,
	key string,
) error {
	if err := client.Del(ctx, key).Err(); err != nil {
		return redis.NewError(err, scope)
	}
	return nil
}

// deleteSetPipelined deletes multiple keys in a single Redis pipeline.
func deleteSetPipelined(
	ctx context.Context,
	scope redis.Scope,
	client redisdriver.Cmdable,
	keys []string,
) error {
	if len(keys) == 0 {
		return nil
	}

	pipe := client.Pipeline()
	for _, key := range keys {
		pipe.Del(ctx, key)
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return redis.NewError(err, scope)
	}

	return nil
}

func getHashMapKeyVals[T any](
	ctx context.Context,
	scope redis.Scope,
	client redisdriver.Cmdable,
	key string,
	parseVal func(string) (T, error),
) ([]uuid.UUID, []T, map[uuid.UUID]T, error) {
	rawMap, err := client.HGetAll(ctx, key).Result()
	if redis.IsCacheMiss(err) {
		return nil, nil, nil, nil
	}
	if err != nil {
		return nil, nil, nil, redis.NewError(err, scope)
	}
	if len(rawMap) == 0 {
		return nil, nil, nil, nil
	}

	keyIDs := make([]uuid.UUID, 0, len(rawMap))
	valMap := make(map[uuid.UUID]T, len(rawMap))

	seenVals := make(map[string]bool)
	valueList := make([]T, 0)

	for k, v := range rawMap {
		sessID, err := uuid.Parse(k)
		if err != nil {
			continue
		}

		if v == "" {
			continue
		}

		parsedVal, err := parseVal(v)
		if err != nil {
			continue
		}

		keyIDs = append(keyIDs, sessID)
		valMap[sessID] = parsedVal

		if !seenVals[v] {
			seenVals[v] = true
			valueList = append(valueList, parsedVal)
		}
	}

	if len(keyIDs) == 0 {
		return nil, nil, nil, nil
	}

	return keyIDs, valueList, valMap, nil
}

func setHashMapKeyVals[T any](
	ctx context.Context,
	scope redis.Scope,
	client redisdriver.Cmdable,
	key string,
	data map[uuid.UUID]T,
	ttl time.Duration,
	formatVal func(T) string,
) error {
	if len(data) == 0 {
		return nil
	}

	args := make([]interface{}, 0, len(data)*2)
	for keyID, val := range data {
		args = append(args, keyID.String(), formatVal(val))
	}

	if ttl > 0 {
		pipe := client.Pipeline()
		pipe.HSet(ctx, key, args...)
		pipe.Expire(ctx, key, ttl)

		if _, err := pipe.Exec(ctx); err != nil {
			return redis.NewError(err, scope)
		}
		return nil
	}

	if err := client.HSet(ctx, key, args...).Err(); err != nil {
		return redis.NewError(err, scope)
	}

	return nil
}

// getBatchHashMaps fetches HGETALL for multiple keys in a single pipeline pass.
func getBatchHashMaps(
	ctx context.Context,
	scope redis.Scope,
	client redisdriver.Cmdable,
	keys []string,
) ([]map[string]string, error) {
	if len(keys) == 0 {
		return nil, nil
	}

	cmds := make([]*redisdriver.MapStringStringCmd, len(keys))
	_, err := client.Pipelined(ctx, func(pipe redisdriver.Pipeliner) error {
		for i, key := range keys {
			cmds[i] = pipe.HGetAll(ctx, key)
		}
		return nil
	})
	if err != nil && !errors.Is(err, redisdriver.Nil) {
		return nil, redis.NewError(err, scope)
	}

	results := make([]map[string]string, len(keys))
	for i, cmd := range cmds {
		res, cmdErr := cmd.Result()
		if cmdErr != nil || len(res) == 0 {
			continue
		}
		results[i] = res
	}

	return results, nil
}
