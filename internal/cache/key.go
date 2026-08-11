package cache

import (
	"context"
	"encoding/json"
	"time"

	"bonfire-api/internal/redis"
)

type KeyCache[K comparable, V any] struct {
	store *redis.Store
	ttl   time.Duration
	keyFn func(K) string
}

func NewKeyCache[K comparable, V any](
	store *redis.Store,
	ttl time.Duration,
	keyFn func(K) string,
) *KeyCache[K, V] {
	return &KeyCache[K, V]{
		store: store,
		ttl:   ttl,
		keyFn: keyFn,
	}
}

func (s *KeyCache[K, V]) Set(ctx context.Context, key K, val *V) error {
	if val == nil {
		return nil
	}
	return s.SetBatch(ctx, map[K]*V{key: val})
}

func (s *KeyCache[K, V]) SetBatch(ctx context.Context, items map[K]*V) error {
	if len(items) == 0 {
		return nil
	}

	return s.store.ExecPipeline(ctx, func(pipeCtx context.Context) error {
		for k, val := range items {
			if val == nil {
				continue
			}
			data, err := json.Marshal(val)
			if err != nil {
				return err
			}
			if err := s.store.Set(pipeCtx, s.keyFn(k), data, s.ttl); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *KeyCache[K, V]) Get(ctx context.Context, key K) (*V, error) {
	found, _, err := s.GetBatch(ctx, []K{key})
	if err != nil {
		return nil, err
	}
	return found[key], nil
}

func (s *KeyCache[K, V]) GetBatch(ctx context.Context, keys []K) (map[K]*V, []K, error) {
	if len(keys) == 0 {
		return make(map[K]*V), nil, nil
	}

	uniqueKeys := make([]K, 0, len(keys))
	seen := make(map[K]struct{}, len(keys))
	for _, k := range keys {
		if _, exists := seen[k]; !exists {
			seen[k] = struct{}{}
			uniqueKeys = append(uniqueKeys, k)
		}
	}

	redisKeys := make([]string, len(uniqueKeys))
	for i, k := range uniqueKeys {
		redisKeys[i] = s.keyFn(k)
	}

	rawItems, err := s.store.MGet(ctx, redisKeys...)
	if err != nil {
		return nil, nil, s.store.Err(err)
	}

	found := make(map[K]*V, len(uniqueKeys))
	var missing []K

	for i, k := range uniqueKeys {
		if i >= len(rawItems) || rawItems[i] == nil {
			missing = append(missing, k)
			continue
		}

		var b []byte
		switch v := rawItems[i].(type) {
		case string:
			b = []byte(v)
		case []byte:
			b = v
		default:
			missing = append(missing, k)
			continue
		}

		if len(b) == 0 {
			missing = append(missing, k)
			continue
		}

		var item V
		if err := json.Unmarshal(b, &item); err != nil {
			missing = append(missing, k)
			continue
		}
		found[k] = &item
	}

	return found, missing, nil
}

func (s *KeyCache[K, V]) Delete(ctx context.Context, key K) error {
	return s.DeleteBatch(ctx, []K{key})
}

func (s *KeyCache[K, V]) DeleteBatch(ctx context.Context, keys []K) error {
	if len(keys) == 0 {
		return nil
	}
	redisKeys := make([]string, len(keys))
	for i, k := range keys {
		redisKeys[i] = s.keyFn(k)
	}
	return s.store.Delete(ctx, redisKeys...)
}
