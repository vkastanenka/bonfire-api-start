package redis

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"time"

	"github.com/redis/go-redis/v9"
)

type Queries struct {
	client redis.Cmdable
}

func New(client redis.Cmdable) *Queries {
	return &Queries{
		client: client,
	}
}

func (q *Queries) getCmd(ctx context.Context) redis.Cmdable {
	return ExtractCmdable(ctx, q.client)
}

// ============================================================================
// String & Key Operations
// ============================================================================

func (q *Queries) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	payload, err := marshalValue(value)
	if err != nil {
		return err
	}

	return q.getCmd(ctx).Set(ctx, key, payload, ttl).Err()
}

func (q *Queries) Get(ctx context.Context, key string, dest interface{}) error {
	if IsPipelined(ctx) {
		return errors.New("cannot execute read operations like Get inside a pipeline callback")
	}

	rawBytes, err := q.getCmd(ctx).Get(ctx, key).Bytes()
	if err != nil {
		return err
	}

	return unmarshalValue(rawBytes, dest)
}

func (q *Queries) MGet(ctx context.Context, keys ...string) ([]interface{}, error) {
	if len(keys) == 0 {
		return nil, nil
	}
	if IsPipelined(ctx) {
		return nil, errors.New("cannot execute MGet inside a pipeline callback")
	}

	return q.getCmd(ctx).MGet(ctx, keys...).Result()
}

func (q *Queries) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return q.getCmd(ctx).Del(ctx, keys...).Err()
}

func (q *Queries) Exists(ctx context.Context, key string) (bool, error) {
	if IsPipelined(ctx) {
		return false, errors.New("cannot execute Exists inside a pipeline callback")
	}

	count, err := q.getCmd(ctx).Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

var incrWithTTLScript = redis.NewScript(`
	local current = redis.call("INCR", KEYS[1])
	if current == 1 then
		redis.call("PEXPIRE", KEYS[1], ARGV[1])
	end
	return current
`)

func (q *Queries) Increment(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	if IsPipelined(ctx) {
		return 0, errors.New("cannot read Increment result inside a pipeline callback")
	}

	ms := ttl.Milliseconds()
	if ms <= 0 {
		ms = 1000
	}

	return incrWithTTLScript.Run(ctx, q.getCmd(ctx), []string{key}, ms).Int64()
}

// ============================================================================
// Hash Operations
// ============================================================================

func (q *Queries) HSet(ctx context.Context, key string, field string, value interface{}) error {
	payload, err := marshalValue(value)
	if err != nil {
		return err
	}

	return q.getCmd(ctx).HSet(ctx, key, field, payload).Err()
}

func (q *Queries) HMSet(ctx context.Context, key string, values map[string]interface{}) error {
	if len(values) == 0 {
		return nil
	}

	args := make([]interface{}, 0, len(values)*2)
	for k, v := range values {
		b, err := marshalValue(v)
		if err != nil {
			return err
		}
		args = append(args, k, b)
	}

	return q.getCmd(ctx).HSet(ctx, key, args...).Err()
}

func (q *Queries) HGet(ctx context.Context, key, field string, dest interface{}) error {
	if IsPipelined(ctx) {
		return errors.New("cannot execute HGet inside a pipeline callback")
	}

	rawBytes, err := q.getCmd(ctx).HGet(ctx, key, field).Bytes()
	if err != nil {
		return err
	}

	return unmarshalValue(rawBytes, dest)
}

func (q *Queries) HDel(ctx context.Context, key string, fields ...string) error {
	if len(fields) == 0 {
		return nil
	}
	return q.getCmd(ctx).HDel(ctx, key, fields...).Err()
}

func (q *Queries) HGetAll(ctx context.Context, key string, dest *map[string]string) error {
	if dest == nil {
		return errors.New("destination map pointer cannot be nil")
	}
	if IsPipelined(ctx) {
		return errors.New("cannot execute HGetAll inside a pipeline callback")
	}

	res, err := q.getCmd(ctx).HGetAll(ctx, key).Result()
	if err != nil {
		return err
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
		return err
	}

	return q.getCmd(ctx).ZAdd(ctx, key, redis.Z{
		Score:  score,
		Member: payload,
	}).Err()
}

func (q *Queries) ZRem(ctx context.Context, key string, members ...interface{}) error {
	if len(members) == 0 {
		return nil
	}

	vals := make([]interface{}, len(members))
	for i, m := range members {
		payload, err := marshalValue(m)
		if err != nil {
			return err
		}
		vals[i] = payload
	}

	return q.getCmd(ctx).ZRem(ctx, key, vals...).Err()
}

func (q *Queries) ZRevRangeByScoreWithScores(ctx context.Context, key string, max, min string, offset, count int64) ([]redis.Z, error) {
	if IsPipelined(ctx) {
		return nil, errors.New("cannot execute ZRevRangeByScoreWithScores inside a pipeline callback")
	}

	return q.getCmd(ctx).ZRangeArgsWithScores(ctx, redis.ZRangeArgs{
		Key:     key,
		Start:   min,
		Stop:    max,
		ByScore: true,
		Rev:     true,
		Offset:  offset,
		Count:   count,
	}).Result()
}

func (q *Queries) ZRemRangeByRank(ctx context.Context, key string, start, stop int64) error {
	return q.getCmd(ctx).ZRemRangeByRank(ctx, key, start, stop).Err()
}

// ============================================================================
// Key Expiry & PubSub Operations
// ============================================================================

func (q *Queries) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return q.getCmd(ctx).Expire(ctx, key, ttl).Err()
}

func (q *Queries) Publish(ctx context.Context, channel string, payload interface{}) error {
	rawBytes, err := marshalValue(payload)
	if err != nil {
		return err
	}

	return q.getCmd(ctx).Publish(ctx, channel, rawBytes).Err()
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
	rv := reflect.ValueOf(dest)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return errors.New("destination target must be a non-nil pointer")
	}

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
