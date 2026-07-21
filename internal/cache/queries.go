package cache

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

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

func (q *Queries) Delete(ctx context.Context, key string) error {
	if err := q.cmd.Del(ctx, key).Err(); err != nil {
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
