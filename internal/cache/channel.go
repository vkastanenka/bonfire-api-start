package cache

import (
	"context"
	"time"

	"bonfire-api/internal/channel"
	"bonfire-api/internal/fields"
	"bonfire-api/internal/redis"

	redisdriver "github.com/redis/go-redis/v9"
)

type ChannelCache struct {
	*KeyCache[fields.ID, Channel]
	ttl time.Duration
}

func NewChannelCache(client redisdriver.Cmdable, ttl time.Duration) *ChannelCache {
	return &ChannelCache{
		KeyCache: NewKeyCache[fields.ID, Channel](client, redis.ScopeChannel, ChannelKey),
		ttl:      ttl,
	}
}

func (c *ChannelCache) Get(ctx context.Context, id fields.ID) (*channel.Channel, error) {
	dto, err := c.KeyCache.Get(ctx, id)
	if err != nil || dto == nil {
		return nil, err
	}

	return dto.ToDomain()
}

func (c *ChannelCache) GetBatch(
	ctx context.Context,
	ids []fields.ID,
) (map[fields.ID]*channel.Channel, []fields.ID, error) {
	dtos, missing, err := c.KeyCache.GetBatch(ctx, ids)
	if err != nil {
		return nil, nil, err
	}

	found := make(map[fields.ID]*channel.Channel, len(dtos))
	for id, dto := range dtos {
		if dto == nil {
			missing = append(missing, id)
			continue
		}

		ch, err := dto.ToDomain()
		if err != nil {
			missing = append(missing, id)
			continue
		}
		found[id] = ch
	}

	return found, missing, nil
}

func (c *ChannelCache) Set(ctx context.Context, ch *channel.Channel) error {
	return c.KeyCache.Set(ctx, ch.ID(), ParseChannel(ch), c.ttl)
}

func (c *ChannelCache) SetBatch(ctx context.Context, channels []*channel.Channel) error {
	dtos := make(map[fields.ID]Channel, len(channels))
	for _, ch := range channels {
		if ch == nil {
			continue
		}
		dtos[ch.ID()] = ParseChannel(ch)
	}

	return c.KeyCache.SetBatch(ctx, dtos, c.ttl)
}
