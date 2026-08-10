package redis

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// ============================================================================
// String & Key Operations
// ============================================================================

func (q *Queries) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	payload, err := marshalValue(value)
	if err != nil {
		return NewError(err, ScopeStore)
	}

	if err := q.cmd.Set(ctx, key, payload, ttl).Err(); err != nil {
		return NewError(err, ScopeStore)
	}
	return nil
}

func (q *Queries) Get(ctx context.Context, key string, dest interface{}) error {
	rawBytes, err := q.cmd.Get(ctx, key).Bytes()
	if err != nil {
		return NewError(err, ScopeStore)
	}

	if err := unmarshalValue(rawBytes, dest); err != nil {
		return NewError(err, ScopeStore)
	}
	return nil
}

func (q *Queries) MGet(ctx context.Context, keys ...string) ([]interface{}, error) {
	if len(keys) == 0 {
		return nil, nil
	}
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
	payload, err := marshalValue(value)
	if err != nil {
		return NewError(err, ScopeStore)
	}

	if err := q.cmd.HSet(ctx, key, field, payload).Err(); err != nil {
		return NewError(err, ScopeStore)
	}
	return nil
}

func (q *Queries) HGet(ctx context.Context, key, field string, dest interface{}) error {
	rawBytes, err := q.cmd.HGet(ctx, key, field).Bytes()
	if err != nil {
		return NewError(err, ScopeStore)
	}

	if err := unmarshalValue(rawBytes, dest); err != nil {
		return NewError(err, ScopeStore)
	}
	return nil
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

// ============================================================================
// Sorted Set (ZSET) Operations
// ============================================================================

func (q *Queries) ZAdd(ctx context.Context, key string, score float64, member interface{}) error {
	payload, err := marshalValue(member)
	if err != nil {
		return NewError(err, ScopeStore)
	}

	err = q.cmd.ZAdd(ctx, key, redis.Z{
		Score:  score,
		Member: payload,
	}).Err()

	if err != nil {
		return NewError(err, ScopeStore)
	}
	return nil
}

func (q *Queries) ZRem(ctx context.Context, key string, members ...interface{}) error {
	if len(members) == 0 {
		return nil
	}

	vals := make([]interface{}, len(members))
	for i, m := range members {
		payload, err := marshalValue(m)
		if err != nil {
			return NewError(err, ScopeStore)
		}
		vals[i] = payload
	}

	if err := q.cmd.ZRem(ctx, key, vals...).Err(); err != nil {
		return NewError(err, ScopeStore)
	}
	return nil
}

func (q *Queries) ZRevRangeByScoreWithScores(ctx context.Context, key string, max, min string, offset, count int64) ([]redis.Z, error) {
	zs, err := q.cmd.ZRangeArgsWithScores(ctx, redis.ZRangeArgs{
		Key:     key,
		Start:   min,
		Stop:    max,
		ByScore: true,
		Rev:     true,
		Offset:  offset,
		Count:   count,
	}).Result()
	if err != nil {
		return nil, NewError(err, ScopeStore)
	}

	return zs, nil
}

func (q *Queries) ZRemRangeByRank(ctx context.Context, key string, start, stop int64) error {
	if err := q.cmd.ZRemRangeByRank(ctx, key, start, stop).Err(); err != nil {
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
	rawBytes, err := marshalValue(payload)
	if err != nil {
		return NewError(err, ScopeEvents)
	}

	if err := q.cmd.Publish(ctx, channel, rawBytes).Err(); err != nil {
		return NewError(err, ScopeEvents)
	}
	return nil
}

// ============================================================================
// Internal Helpers
// ============================================================================

func marshalValue(v interface{}) ([]byte, error) {
	switch val := v.(type) {
	case []byte:
		return val, nil
	case string:
		return []byte(val), nil
	default:
		return json.Marshal(v)
	}
}

func unmarshalValue(raw []byte, dest interface{}) error {
	switch d := dest.(type) {
	case *string:
		*d = string(raw)
		return nil
	case *[]byte:
		*d = raw
		return nil
	default:
		return json.Unmarshal(raw, dest)
	}
}
