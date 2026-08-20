package cache

import (
	"context"
	"time"

	"bonfire-api/internal/fields"
	"bonfire-api/internal/presence"
	"bonfire-api/internal/redis"

	redisdriver "github.com/redis/go-redis/v9"
)

func presenceKey(userID fields.ID) string {
	return "user:" + userID.String() + ":presence"
}

type PresenceCache struct {
	*ScopeCache[fields.ID, int16]
	ttl time.Duration
}

func NewPresenceCache(client redisdriver.Cmdable, ttl time.Duration) *PresenceCache {
	return &PresenceCache{
		ScopeCache: NewScopeCache[fields.ID, int16](client, redis.ScopePresence, presenceKey),
		ttl:        ttl,
	}
}

func (c *PresenceCache) Get(ctx context.Context, userID fields.ID) (presence.Presence, error) {
	val, err := c.ScopeCache.Get(ctx, userID)
	if err != nil {
		return presence.New(presence.PresenceUnknown), err
	}
	if val == nil {
		return presence.New(presence.PresenceOffline), nil
	}

	p, err := presence.Parse(*val)
	if err != nil {
		return presence.New(presence.PresenceOffline), nil
	}

	return p, nil
}

func (c *PresenceCache) GetBatch(
	ctx context.Context,
	userIDs []fields.ID,
) (map[fields.ID]presence.Presence, error) {
	dtos, _, err := c.ScopeCache.GetBatch(ctx, userIDs)
	if err != nil {
		return nil, err
	}

	result := make(map[fields.ID]presence.Presence, len(userIDs))
	for _, id := range userIDs {
		val, found := dtos[id]
		if !found || val == nil {
			result[id] = presence.New(presence.PresenceOffline)
			continue
		}

		p, err := presence.Parse(*val)
		if err != nil {
			result[id] = presence.New(presence.PresenceOffline)
			continue
		}

		result[id] = p
	}

	return result, nil
}

func (c *PresenceCache) Set(ctx context.Context, userID fields.ID, p presence.Presence) error {
	return c.ScopeCache.Set(ctx, userID, p.Int16(), c.ttl)
}

func (c *PresenceCache) SetBatch(ctx context.Context, items map[fields.ID]presence.Presence) error {
	if len(items) == 0 {
		return nil
	}

	dtos := make(map[fields.ID]int16, len(items))
	for id, p := range items {
		dtos[id] = p.Int16()
	}

	return c.ScopeCache.SetBatch(ctx, dtos, c.ttl)
}
