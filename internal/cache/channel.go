package cache

import (
	"context"
	"encoding/json"
	"time"

	"bonfire-api/internal/channel"
	"bonfire-api/internal/errs"
	"bonfire-api/internal/fields"
	"bonfire-api/internal/redis"

	redisdriver "github.com/redis/go-redis/v9"
)

type ChannelCache struct {
	client redisdriver.Cmdable
	scope  redis.Scope
	ttl    time.Duration
}

func NewChannelCache(
	client redisdriver.Cmdable,
	scope redis.Scope,
	ttl time.Duration,
) *ChannelCache {
	return &ChannelCache{
		client: client,
		scope:  scope,
		ttl:    ttl,
	}
}

func (c *ChannelCache) Get(ctx context.Context, id fields.ID) (*channel.Channel, error) {
	redisKey := ChannelKey(id)

	data, err := c.client.Get(ctx, redisKey).Bytes()
	if redis.IsCacheMiss(err) {
		return nil, nil
	}
	if err != nil {
		return nil, redis.NewError(err, c.scope)
	}

	var dto Channel
	if err := json.Unmarshal(data, &dto); err != nil {
		return nil, errs.Internal("Failed to unmarshal cached channel.").
			Meta("scope", c.scope.String()).
			Wrap(err)
	}

	return dto.ToDomain()
}

func (c *ChannelCache) GetBatch(
	ctx context.Context,
	ids []fields.ID,
) (map[fields.ID]*channel.Channel, []fields.ID, error) {
	found := make(map[fields.ID]*channel.Channel, len(ids))
	var missing []fields.ID

	for i := 0; i < len(ids); i += KeyMaxBatchSize {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}

		end := min(i+KeyMaxBatchSize, len(ids))
		chunk := ids[i:end]

		redisKeys := make([]string, len(chunk))
		for j, id := range chunk {
			redisKeys[j] = ChannelKey(id)
		}

		vals, err := c.client.MGet(ctx, redisKeys...).Result()
		if err != nil {
			return nil, nil, redis.NewError(err, c.scope)
		}

		for j, raw := range vals {
			id := chunk[j]

			if raw == nil {
				missing = append(missing, id)
				continue
			}

			var data []byte
			switch v := raw.(type) {
			case string:
				if v == "" {
					missing = append(missing, id)
					continue
				}
				data = []byte(v)
			case []byte:
				if len(v) == 0 {
					missing = append(missing, id)
					continue
				}
				data = v
			default:
				missing = append(missing, id)
				continue
			}

			var dto Channel
			if err := json.Unmarshal(data, &dto); err != nil {
				return nil, nil, errs.Internal("Failed to unmarshal cached channel batch item.").
					Meta("scope", c.scope.String()).
					Wrap(err)
			}

			ch, err := dto.ToDomain()
			if err != nil {
				missing = append(missing, id)
				continue
			}

			found[id] = ch
		}
	}

	if len(missing) > 0 {
		seen := make(map[fields.ID]struct{}, len(missing))
		result := make([]fields.ID, 0, len(missing))

		for _, id := range missing {
			if _, inFound := found[id]; inFound {
				continue
			}
			if _, inSeen := seen[id]; !inSeen {
				seen[id] = struct{}{}
				result = append(result, id)
			}
		}
		missing = result
	}

	return found, missing, nil
}

func (c *ChannelCache) Set(ctx context.Context, ch *channel.Channel) error {
	redisKey := ChannelKey(ch.ID())
	dto := ParseChannel(ch)

	bytes, err := json.Marshal(dto)
	if err != nil {
		return errs.Internal("Failed to marshal channel json.").
			Meta("scope", c.scope.String()).
			Wrap(err)
	}

	if err := c.client.Set(ctx, redisKey, bytes, c.ttl).Err(); err != nil {
		return redis.NewError(err, c.scope)
	}

	return nil
}

func (c *ChannelCache) SetBatch(ctx context.Context, channels []*channel.Channel) error {
	validChannels := make([]*channel.Channel, 0, len(channels))
	for _, ch := range channels {
		if ch != nil {
			validChannels = append(validChannels, ch)
		}
	}

	for i := 0; i < len(validChannels); i += KeyMaxBatchSize {
		if err := ctx.Err(); err != nil {
			return err
		}

		end := min(i+KeyMaxBatchSize, len(validChannels))
		chunk := validChannels[i:end]

		pipe := c.client.Pipeline()
		for _, ch := range chunk {
			bytes, err := json.Marshal(ParseChannel(ch))
			if err != nil {
				return errs.Internal("Failed to marshal channel json.").
					Meta("scope", c.scope.String()).
					Wrap(err)
			}
			pipe.Set(ctx, ChannelKey(ch.ID()), bytes, c.ttl)
		}

		if _, err := pipe.Exec(ctx); err != nil {
			return redis.NewError(err, c.scope)
		}
	}

	return nil
}

func (c *ChannelCache) Delete(ctx context.Context, id fields.ID) error {
	if err := c.client.Del(ctx, ChannelKey(id)).Err(); err != nil {
		return redis.NewError(err, c.scope)
	}

	return nil
}

func (c *ChannelCache) DeleteBatch(ctx context.Context, ids []fields.ID) error {
	for i := 0; i < len(ids); i += KeyMaxBatchSize {
		if err := ctx.Err(); err != nil {
			return err
		}

		end := min(i+KeyMaxBatchSize, len(ids))
		chunk := ids[i:end]

		redisKeys := make([]string, len(chunk))
		for j, id := range chunk {
			redisKeys[j] = ChannelKey(id)
		}

		if err := c.client.Del(ctx, redisKeys...).Err(); err != nil {
			return redis.NewError(err, c.scope)
		}
	}

	return nil
}

func (c *ChannelCache) AddMemberIDs(ctx context.Context, channelID fields.ID, userIDs []fields.ID) error {
	members := make([]any, 0, len(userIDs))
	for _, id := range userIDs {
		if !id.IsZero() {
			members = append(members, id.String())
		}
	}
	if len(members) == 0 {
		return nil
	}

	key := ChannelMemberIDsKey(channelID)

	pipe := c.client.Pipeline()
	pipe.SAdd(ctx, key, members...)
	pipe.Expire(ctx, key, c.ttl)

	if _, err := pipe.Exec(ctx); err != nil {
		return redis.NewError(err, redis.ScopeChannel)
	}

	return nil
}

func (c *ChannelCache) RemoveMemberIDs(ctx context.Context, channelID fields.ID, userIDs []fields.ID) error {
	members := make([]any, 0, len(userIDs))
	for _, id := range userIDs {
		if !id.IsZero() {
			members = append(members, id.String())
		}
	}
	if len(members) == 0 {
		return nil
	}

	key := ChannelMemberIDsKey(channelID)

	if err := c.client.SRem(ctx, key, members...).Err(); err != nil {
		return redis.NewError(err, redis.ScopeChannel)
	}

	return nil
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

func (c *ChannelCache) SetNew(ctx context.Context, ch *channel.Channel) error {
	if ch == nil {
		return nil
	}

	dto := ParseChannel(ch)
	bytes, err := json.Marshal(dto)
	if err != nil {
		return errs.Internal("Failed to marshal channel json.").
			Meta("scope", c.scope.String()).
			Wrap(err)
	}

	pipe := c.client.Pipeline()
	pipe.Set(ctx, ChannelKey(ch.ID()), bytes, c.ttl)
	pipe.Set(ctx, ChannelLoadedKey(ch.ID()), "1", c.ttl)

	if _, err := pipe.Exec(ctx); err != nil {
		return redis.NewError(err, c.scope)
	}

	return nil
}
