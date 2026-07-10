package cache

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

func (m *manager) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
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

	if err := m.client.Set(ctx, key, bytes, ttl).Err(); err != nil {
		return NewError(err, ScopeStore)
	}
	return nil
}

func (m *manager) Get(ctx context.Context, key string, dest interface{}) error {
	bytes, err := m.client.Get(ctx, key).Bytes()
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

func (m *manager) MGet(ctx context.Context, keys ...string) ([]interface{}, error) {
	values, err := m.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, NewError(err, ScopeStore)
	}
	return values, nil
}

func (m *manager) Delete(ctx context.Context, key string) error {
	if err := m.client.Del(ctx, key).Err(); err != nil {
		return NewError(err, ScopeStore)
	}
	return nil
}

func (m *manager) Exists(ctx context.Context, key string) (bool, error) {
	count, err := m.client.Exists(ctx, key).Result()
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

func (m *manager) Increment(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	seconds := int64(ttl.Seconds())

	result, err := incrWithTTLScript.Run(ctx, m.client, []string{key}, seconds).Result()
	if err != nil {
		return 0, NewError(err, ScopeStore)
	}

	if val, ok := result.(int64); ok {
		return val, nil
	}

	return 0, NewError(errors.New("redis script returned unexpected type"), ScopeStore)
}
