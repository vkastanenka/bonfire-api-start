package cache

import (
	"context"
	"time"

	"bonfire-api/internal/presence"
	"bonfire-api/internal/redis"

	"github.com/google/uuid"
	redisdriver "github.com/redis/go-redis/v9"
)

var (
	userPresenceTTL = 90 * time.Second
)

func userPresenceKey(id uuid.UUID) string { return "{user:" + id.String() + "}:presence" }

type PresenceCache struct {
	client redisdriver.Cmdable
}

func NewPresenceCache(client redisdriver.Cmdable) *PresenceCache {
	return &PresenceCache{client: client}
}

func (c *PresenceCache) Get(ctx context.Context, userID uuid.UUID) (presence.Presence, error) {
	data, found, err := getKey(ctx, redis.ScopePresence, c.client, userPresenceKey(userID))
	if err != nil {
		return presence.PresenceOffline, err
	}
	if !found {
		return presence.PresenceOffline, nil
	}

	val, err := presence.ParseIntBytes(data)
	if err != nil {
		return presence.PresenceOffline, nil
	}

	return val, nil
}

func (c *PresenceCache) Set(ctx context.Context, userID uuid.UUID, p presence.Presence) error {
	if err := c.client.Set(ctx, userPresenceKey(userID), p.Int(), userPresenceTTL).Err(); err != nil {
		return redis.NewError(err, redis.ScopePresence)
	}
	return nil
}

func (c *PresenceCache) Delete(ctx context.Context, id uuid.UUID) error {
	if err := c.client.Del(ctx, userPresenceKey(id)).Err(); err != nil {
		return redis.NewError(err, redis.ScopePresence)
	}
	return nil
}

func (c *PresenceCache) GetBatch(
	ctx context.Context,
	userIDs []uuid.UUID,
) (map[uuid.UUID]presence.Presence, error) {
	result := make(map[uuid.UUID]presence.Presence, len(userIDs))
	if len(userIDs) == 0 {
		return result, nil
	}

	for i := 0; i < len(userIDs); i += maxBatchSize {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		end := min(i+maxBatchSize, len(userIDs))
		chunk := userIDs[i:end]

		redisKeys := make([]string, len(chunk))
		for j, id := range chunk {
			redisKeys[j] = userPresenceKey(id)
		}

		vals, err := getBatchKeys(ctx, redis.ScopePresence, c.client, redisKeys)
		if err != nil {
			return nil, err
		}

		for j, raw := range vals {
			id := chunk[j]

			data, ok := toBytes(raw)
			if !ok {
				result[id] = presence.PresenceOffline
				continue
			}

			val, err := presence.ParseIntBytes(data)
			if err != nil {
				result[id] = presence.PresenceOffline
				continue
			}

			result[id] = val
		}
	}

	return result, nil
}
