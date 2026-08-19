package cache

import (
	"context"
	"encoding/json"
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

func (c *ChannelCache) SetMemberIDs(ctx context.Context, channelID fields.ID, memberIDs []fields.ID) error {
	if channelID.IsZero() || len(memberIDs) == 0 {
		return nil
	}

	key := ChannelMemberIDsKey(channelID)

	// Convert []fields.ID to string/byte slice representation for Redis
	rawIDs := make([]string, 0, len(memberIDs))
	for _, id := range memberIDs {
		if !id.IsZero() {
			rawIDs = append(rawIDs, id.String())
		}
	}

	if len(rawIDs) == 0 {
		return nil
	}

	// Use Redis SADD / RPUSH or JSON/SET depending on your storage strategy.
	// Example using JSON/String serialization matching KeyCache TTL:
	data, err := json.Marshal(rawIDs)
	if err != nil {
		return err
	}

	return c.KeyCache.client.Set(ctx, key, data, c.ttl).Err()
}

func (c *ChannelCache) IsLoaded(ctx context.Context, channelID fields.ID) bool {
	val, err := c.client.Exists(ctx, ChannelLoadedKey(channelID)).Result()
	return err == nil && val > 0
}

func (c *ChannelCache) SetLoaded(ctx context.Context, channelID fields.ID) error {
	key := ChannelLoadedKey(channelID)
	if err := c.client.Set(ctx, key, "1", c.ttl).Err(); err != nil {
		return redis.NewError(err, redis.ScopeMessage)
	}
	return nil
}

// func (c *ChannelCache) GetMemberIDs(ctx context.Context, channelID fields.ID) ([]fields.ID, error) {
// 	if channelID.IsZero() {
// 		return nil, nil
// 	}

// 	key := ChannelMemberIDsKey(channelID)
// 	val, err := c.KeyCache.client.Get(ctx, key).Result()
// 	if err == redisdriver.Nil {
// 		return nil, nil // Cache miss
// 	} else if err != nil {
// 		return nil, err
// 	}

// 	var rawIDs []string
// 	if err := json.Unmarshal([]byte(val), &rawIDs); err != nil {
// 		return nil, err
// 	}

// 	memberIDs, err := fields.ParseIDs("member_id", fields.parseUUIDs(rawIDs))
// 	if err != nil {
// 		return nil, err
// 	}

// 	return memberIDs, nil
// }
