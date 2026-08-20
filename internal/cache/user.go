package cache

import (
	"context"
	"strconv"
	"time"

	"bonfire-api/internal/fields"
	"bonfire-api/internal/presence"
	"bonfire-api/internal/redis"

	redisdriver "github.com/redis/go-redis/v9"
)

func userPresenceKey(userID fields.ID) string {
	return "user:" + userID.String() + ":presence"
}

type UserCache struct {
	client redisdriver.Cmdable
	scope  redis.Scope
	ttl    time.Duration
}

func NewUserCache(client redisdriver.Cmdable, scope redis.Scope, ttl time.Duration) *UserCache {
	return &UserCache{
		client: client,
		scope:  scope,
		ttl:    ttl,
	}
}

func (c *UserCache) GetPresence(ctx context.Context, userID fields.ID) (presence.Presence, error) {
	val, err := c.client.Get(ctx, userPresenceKey(userID)).Uint64()
	if redis.IsCacheMiss(err) {
		return presence.New(presence.PresenceOffline), nil
	}
	if err != nil {
		return presence.New(presence.PresenceUnknown), redis.NewError(err, c.scope)
	}

	p, err := presence.Parse(uint8(val))
	if err != nil {
		return presence.New(presence.PresenceOffline), nil
	}

	return p, nil
}

func (c *UserCache) GetBatchPresence(
	ctx context.Context,
	userIDs []fields.ID,
) (map[fields.ID]presence.Presence, error) {
	result := make(map[fields.ID]presence.Presence, len(userIDs))
	if len(userIDs) == 0 {
		return result, nil
	}

	redisKeys := make([]string, len(userIDs))
	for j, id := range userIDs {
		redisKeys[j] = userPresenceKey(id)
	}

	vals, err := c.client.MGet(ctx, redisKeys...).Result()
	if err != nil {
		return nil, redis.NewError(err, c.scope)
	}

	for j, raw := range vals {
		userID := userIDs[j]

		valStr, ok := raw.(string)
		if !ok || raw == nil {
			result[userID] = presence.New(presence.PresenceOffline)
			continue
		}

		val, parseErr := strconv.ParseUint(valStr, 10, 8)
		if parseErr != nil {
			result[userID] = presence.New(presence.PresenceOffline)
			continue
		}

		p, parseErr := presence.Parse(uint8(val))
		if parseErr != nil {
			result[userID] = presence.New(presence.PresenceOffline)
			continue
		}

		result[userID] = p
	}

	return result, nil
}

func (c *UserCache) SetPresence(ctx context.Context, userID fields.ID, p presence.Presence) error {
	if err := c.client.Set(ctx, userPresenceKey(userID), p.Uint8(), c.ttl).Err(); err != nil {
		return redis.NewError(err, c.scope)
	}
	return nil
}

func (c *UserCache) SetBatchPresence(ctx context.Context, items map[fields.ID]presence.Presence) error {
	if len(items) == 0 {
		return nil
	}

	pipe := c.client.Pipeline()
	for userID, p := range items {
		pipe.Set(ctx, userPresenceKey(userID), p.Uint8(), c.ttl)
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return redis.NewError(err, c.scope)
	}

	return nil
}
