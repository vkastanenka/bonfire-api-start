package cache

import (
	"context"
	"time"

	"bonfire-api/internal/channel"
	"bonfire-api/internal/redis"

	"github.com/google/uuid"
	redisdriver "github.com/redis/go-redis/v9"
)

const (
	channelTTL      = 24 * time.Hour
	userChannelsTTL = 24 * time.Hour
)

func channelKey(channelID uuid.UUID) string {
	return "{channel:" + channelID.String() + "}"
}

func userChannelsKey(id uuid.UUID) string {
	return "{user:" + id.String() + "}:channels"
}

type ChannelCache struct {
	client redisdriver.Cmdable
}

func NewChannelCache(client redisdriver.Cmdable) *ChannelCache {
	return &ChannelCache{
		client: client,
	}
}

func (c *ChannelCache) Get(ctx context.Context, id uuid.UUID) (*channel.Channel, error) {
	return getAndUnmarshal(ctx, redis.ScopeChannel, c.client, channelKey(id), unmarshalChannel)
}

func (c *ChannelCache) Set(ctx context.Context, ch *channel.Channel) error {
	return marshalAndSet(ctx, redis.ScopeChannel, c.client, channelKey(ch.ID), ch, channelTTL, marshalChannel)
}

func (c *ChannelCache) Delete(ctx context.Context, id uuid.UUID) error {
	if err := c.client.Del(ctx, channelKey(id)).Err(); err != nil {
		return redis.NewError(err, redis.ScopeChannel)
	}
	return nil
}

func (c *ChannelCache) GetBatch(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*channel.Channel, []uuid.UUID, error) {
	return getAndUnmarshalBatch(ctx, redis.ScopeChannel, c.client, ids, channelKey, unmarshalChannel)
}

func (c *ChannelCache) SetBatch(ctx context.Context, channels map[uuid.UUID]*channel.Channel) error {
	return marshalAndSetBatch(ctx, redis.ScopeChannel, c.client, channels, channelKey, channelTTL, marshalChannel)
}

func (c *ChannelCache) CreateGroup(ctx context.Context, ch *channel.Channel, members []*channel.Member) error {
	chBytes, err := marshalChannel(ch)
	if err != nil {
		return err
	}

	memberMap := make(map[string]any, len(members))
	for _, m := range members {
		if m == nil || m.UserID == uuid.Nil {
			continue
		}
		mBytes, err := marshalMember(m)
		if err != nil {
			return err
		}
		memberMap[m.UserID.String()] = mBytes
	}

	pipe := c.client.Pipeline()

	pipe.Set(ctx, channelKey(ch.ID), chBytes, channelTTL)

	if len(memberMap) > 0 {
		pipe.HSet(ctx, channelMembersKey(ch.ID), memberMap)
	}

	channelIDStr := ch.ID.String()
	for _, m := range members {
		if m == nil || m.UserID == uuid.Nil {
			continue
		}

		var score float64
		if m.PinnedAt != nil && !m.PinnedAt.IsZero() {
			score = 1e12 + float64(m.PinnedAt.Unix())
		} else {
			score = float64(ch.CreatedAt.Unix())
		}

		pipe.ZAdd(ctx, userChannelsKey(m.UserID), redisdriver.Z{
			Score:  score,
			Member: channelIDStr,
		})
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return redis.NewError(err, redis.ScopeChannel)
	}

	return nil
}

func (c *ChannelCache) DeleteGroup(ctx context.Context, channelID uuid.UUID, memberIDs []uuid.UUID) error {
	pipe := c.client.Pipeline()

	pipe.Del(ctx, channelKey(channelID))
	pipe.Del(ctx, channelMembersKey(channelID))
	pipe.Del(ctx, channelMessagesKey(channelID))

	channelIDStr := channelID.String()
	for _, userID := range memberIDs {
		if userID != uuid.Nil {
			pipe.ZRem(ctx, userChannelsKey(userID), channelIDStr)
		}
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return redis.NewError(err, redis.ScopeChannel)
	}

	return nil
}

func (c *ChannelCache) GetUserChannelIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, bool, error) {
	if userID == uuid.Nil {
		return nil, false, nil
	}

	rawIDs, err := c.client.ZRevRange(ctx, userChannelsKey(userID), 0, -1).Result()
	if err != nil {
		if redis.IsCacheMiss(err) {
			return nil, false, nil
		}
		return nil, false, redis.NewError(err, redis.ScopeUser)
	}

	if len(rawIDs) == 0 {
		return nil, false, nil
	}

	ids := make([]uuid.UUID, 0, len(rawIDs))
	for _, raw := range rawIDs {
		id, err := uuid.Parse(raw)
		if err == nil {
			ids = append(ids, id)
		}
	}

	return ids, true, nil
}

func (c *ChannelCache) SetUserChannelIDs(ctx context.Context, userID uuid.UUID, members []*channel.Member) error {
	if userID == uuid.Nil || len(members) == 0 {
		return nil
	}

	pipe := c.client.Pipeline()
	zKey := userChannelsKey(userID)

	pipe.Del(ctx, zKey)

	for _, m := range members {
		if m == nil || m.ChannelID == uuid.Nil {
			continue
		}

		var score float64
		if m.PinnedAt != nil && !m.PinnedAt.IsZero() {
			score = 1e12 + float64(m.PinnedAt.Unix())
		} else {
			score = float64(m.CreatedAt.Unix())
		}

		pipe.ZAdd(ctx, zKey, redisdriver.Z{
			Score:  score,
			Member: m.ChannelID.String(),
		})
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return redis.NewError(err, redis.ScopeUser)
	}
	return nil
}

func (c *ChannelCache) RemoveUserChannelID(ctx context.Context, userID, channelID uuid.UUID) error {
	if err := c.client.ZRem(ctx, userChannelsKey(userID), channelID.String()).Err(); err != nil {
		return redis.NewError(err, redis.ScopeUser)
	}

	return nil
}
