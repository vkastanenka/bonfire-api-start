package redis

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// ============================================================================
// Basic String & Key Operations
// ============================================================================

func (q *Queries) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	var bytes []byte
	var err error

	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		bytes, err = json.Marshal(value)
		if err != nil {
			return NewError(err, ScopeStore)
		}
	}

	if err := q.cmd.Set(ctx, key, bytes, ttl).Err(); err != nil {
		return NewError(err, ScopeStore)
	}
	return nil
}

// Portfolio Pro-Tip: Mention in your README or inline docs that read paths use golang.org/x/sync/singleflight
// combined with Redis SetNX to prevent thundering herd problems during invalidation bursts.
func (q *Queries) SetNX(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
	ok, err := q.cmd.SetNX(ctx, key, value, ttl).Result()
	if err != nil {
		return false, NewError(err, ScopeStore)
	}
	return ok, nil
}

func (q *Queries) Get(ctx context.Context, key string, dest interface{}) error {
	bytes, err := q.cmd.Get(ctx, key).Bytes()
	if IsNotFoundError(err) {
		return NewError(ErrNotFound, ScopeStore)
	}
	if err != nil {
		return NewError(err, ScopeStore)
	}

	switch d := dest.(type) {
	case *string:
		*d = string(bytes)
		return nil
	case *[]byte:
		*d = bytes
		return nil
	default:
		if err := json.Unmarshal(bytes, dest); err != nil {
			return NewError(err, ScopeStore)
		}
		return nil
	}
}

func (q *Queries) MGet(ctx context.Context, keys ...string) ([]interface{}, error) {
	values, err := q.cmd.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, NewError(err, ScopeStore)
	}
	return values, nil
}

func (q *Queries) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	if err := q.cmd.Del(ctx, keys...).Err(); err != nil {
		return NewError(err, ScopeStore)
	}
	return nil
}

func (q *Queries) Exists(ctx context.Context, key string) (bool, error) {
	count, err := q.cmd.Exists(ctx, key).Result()
	if err != nil {
		return false, NewError(err, ScopeStore)
	}
	return count > 0, nil
}

var incrWithTTLScript = redis.NewScript(`
	local current = redis.call("INCR", KEYS[1])
	if current == 1 then
		redis.call("EXPIRE", KEYS[1], ARGV[1])
	end
	return current
`)

func (q *Queries) Increment(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	seconds := int64(ttl.Seconds())

	result, err := incrWithTTLScript.Run(ctx, q.cmd, []string{key}, seconds).Result()
	if err != nil {
		return 0, NewError(err, ScopeStore)
	}

	if val, ok := result.(int64); ok {
		return val, nil
	}

	return 0, NewError(errors.New("redis script returned unexpected type"), ScopeStore)
}

// ============================================================================
// Hash Operations
// ============================================================================

func (q *Queries) HSet(ctx context.Context, key string, field string, value interface{}) error {
	var val interface{}

	switch v := value.(type) {
	case []byte:
		val = string(v)
	case string:
		val = v
	default:
		bytes, err := json.Marshal(value)
		if err != nil {
			return NewError(err, ScopeStore)
		}
		val = string(bytes)
	}

	if err := q.cmd.HSet(ctx, key, field, val).Err(); err != nil {
		return NewError(err, ScopeStore)
	}
	return nil
}

// HGet fetches a single field from a Hash and unmarshals it into dest.
func (q *Queries) HGet(ctx context.Context, key, field string, dest interface{}) error {
	bytes, err := q.cmd.HGet(ctx, key, field).Bytes()
	if IsNotFoundError(err) {
		return NewError(ErrNotFound, ScopeStore)
	}
	if err != nil {
		return NewError(err, ScopeStore)
	}

	switch d := dest.(type) {
	case *string:
		*d = string(bytes)
		return nil
	case *[]byte:
		*d = bytes
		return nil
	default:
		if err := json.Unmarshal(bytes, dest); err != nil {
			return NewError(err, ScopeStore)
		}
		return nil
	}
}

func (q *Queries) HDel(ctx context.Context, key string, fields ...string) error {
	if len(fields) == 0 {
		return nil
	}
	if err := q.cmd.HDel(ctx, key, fields...).Err(); err != nil {
		return NewError(err, ScopeStore)
	}
	return nil
}

func (q *Queries) HGetAll(ctx context.Context, key string, dest *map[string]string) error {
	res, err := q.cmd.HGetAll(ctx, key).Result()
	if err != nil {
		return NewError(err, ScopeStore)
	}

	*dest = res
	return nil
}

// HIncrBy atomically increments an integer field inside a Redis Hash.
func (q *Queries) HIncrBy(ctx context.Context, key, field string, incr int64) (int64, error) {
	val, err := q.cmd.HIncrBy(ctx, key, field, incr).Result()
	if err != nil {
		return 0, NewError(err, ScopeStore)
	}
	return val, nil
}

// HMGet retrieves specific field values from a Redis Hash in a single call.
func (q *Queries) HMGet(ctx context.Context, key string, fields ...string) ([]interface{}, error) {
	if len(fields) == 0 {
		return nil, nil
	}

	values, err := q.cmd.HMGet(ctx, key, fields...).Result()
	if err != nil {
		return nil, NewError(err, ScopeStore)
	}
	return values, nil
}

// ============================================================================
// Sorted Set (ZSET) Operations
// ============================================================================

// ZAdd adds a member with a score (e.g., timestamp or snowflake ID) to a sorted set.
func (q *Queries) ZAdd(ctx context.Context, key string, score float64, member interface{}) error {
	var val interface{}

	switch v := member.(type) {
	case []byte:
		val = string(v)
	case string:
		val = v
	default:
		bytes, err := json.Marshal(member)
		if err != nil {
			return NewError(err, ScopeStore)
		}
		val = string(bytes)
	}

	err := q.cmd.ZAdd(ctx, key, redis.Z{
		Score:  score,
		Member: val,
	}).Err()

	if err != nil {
		return NewError(err, ScopeStore)
	}
	return nil
}

// ZRem removes one or more members from a sorted set.
func (q *Queries) ZRem(ctx context.Context, key string, members ...interface{}) error {
	if len(members) == 0 {
		return nil
	}

	vals := make([]interface{}, len(members))
	for i, m := range members {
		switch v := m.(type) {
		case []byte:
			vals[i] = string(v)
		case string:
			vals[i] = v
		default:
			bytes, err := json.Marshal(m)
			if err != nil {
				return NewError(err, ScopeStore)
			}
			vals[i] = string(bytes)
		}
	}

	if err := q.cmd.ZRem(ctx, key, vals...).Err(); err != nil {
		return NewError(err, ScopeStore)
	}
	return nil
}

// ZCard returns the number of elements in a sorted set.
func (q *Queries) ZCard(ctx context.Context, key string) (int64, error) {
	count, err := q.cmd.ZCard(ctx, key).Result()
	if err != nil {
		return 0, NewError(err, ScopeStore)
	}
	return count, nil
}

// ZRangeByScore fetches members in a score range and unmarshals them into the target slice pointer (dest).
func (q *Queries) ZRangeByScore(ctx context.Context, key string, min, max string, offset, count int64, dest interface{}) error {
	res, err := q.cmd.ZRangeArgs(ctx, redis.ZRangeArgs{
		Key:     key,
		Start:   min,
		Stop:    max,
		ByScore: true,
		Offset:  offset,
		Count:   count,
	}).Result()
	if err != nil {
		return NewError(err, ScopeStore)
	}

	switch d := dest.(type) {
	case *[]string:
		*d = res
		return nil
	default:
		rawJSON := "["
		for i, item := range res {
			if i > 0 {
				rawJSON += ","
			}
			rawJSON += item
		}
		rawJSON += "]"

		if err := json.Unmarshal([]byte(rawJSON), dest); err != nil {
			return NewError(err, ScopeStore)
		}
		return nil
	}
}

// ZRevRangeByScoreWithScores fetches members and their scores in descending score order.
func (q *Queries) ZRevRangeByScoreWithScores(ctx context.Context, key string, max, min string, offset, count int64) ([]redis.Z, error) {
	zs, err := q.cmd.ZRangeArgsWithScores(ctx, redis.ZRangeArgs{
		Key:     key,
		Start:   min,
		Stop:    max,
		ByScore: true,
		Rev:     true, // Highest scores (newest) first
		Offset:  offset,
		Count:   count,
	}).Result()
	if err != nil {
		return nil, NewError(err, ScopeChannel)
	}

	return zs, nil
}

// ZRemRangeByRank removes members within the given rank range (e.g., keeping only top N recent items).
func (q *Queries) ZRemRangeByRank(ctx context.Context, key string, start, stop int64) error {
	if err := q.cmd.ZRemRangeByRank(ctx, key, start, stop).Err(); err != nil {
		return NewError(err, ScopeStore)
	}
	return nil
}

// ============================================================================
// Unordered Set Operations
// ============================================================================

// SAdd adds unique members to an unordered Set (useful for online presence or room member lists).
func (q *Queries) SAdd(ctx context.Context, key string, members ...interface{}) error {
	if len(members) == 0 {
		return nil
	}
	if err := q.cmd.SAdd(ctx, key, members...).Err(); err != nil {
		return NewError(err, ScopeStore)
	}
	return nil
}

// ============================================================================
// Key Expiry & PubSub Operations
// ============================================================================

func (q *Queries) Expire(ctx context.Context, key string, ttl time.Duration) error {
	if err := q.cmd.Expire(ctx, key, ttl).Err(); err != nil {
		return NewError(err, ScopeStore)
	}
	return nil
}

func (q *Queries) Publish(ctx context.Context, channel string, payload interface{}) error {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return NewError(err, ScopeEvents)
	}

	if err := q.cmd.Publish(ctx, channel, bytes).Err(); err != nil {
		return NewError(err, ScopeEvents)
	}
	return nil
}
