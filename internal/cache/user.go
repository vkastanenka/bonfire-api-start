package cache

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"bonfire-api/internal/channel"
	"bonfire-api/internal/redis"
	"bonfire-api/internal/user"

	"github.com/google/uuid"
	redisdriver "github.com/redis/go-redis/v9"
)

var (
	userTTL         = 24 * time.Hour
	userChannelsTTL = 24 * time.Hour
)

const (
	emptySetSentinel = "__empty__"
)

func userKey(id uuid.UUID) string         { return "{user:" + id.String() + "}" }
func userPresenceKey(id uuid.UUID) string { return "{user:" + id.String() + "}:presence" }
func userChannelsKey(id uuid.UUID) string { return "{user:" + id.String() + "}:channels" }

type UserCache struct {
	client redisdriver.Cmdable
}

func NewUserCache(client redisdriver.Cmdable) *UserCache {
	return &UserCache{client: client}
}

func (c *UserCache) Get(ctx context.Context, id uuid.UUID) (*user.User, error) {
	return getAndUnmarshal(ctx, redis.ScopeUser, c.client, userKey(id), unmarshalUser)
}

func (c *UserCache) Set(ctx context.Context, usr *user.User) error {
	return marshalAndSet(ctx, redis.ScopeUser, c.client, userKey(usr.ID), usr, userTTL, marshalUser)
}

func (c *UserCache) Delete(ctx context.Context, id uuid.UUID) error {
	if err := c.client.Del(ctx, userKey(id)).Err(); err != nil {
		return redis.NewError(err, redis.ScopeUser)
	}
	return nil
}

func (c *UserCache) GetBatch(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*user.User, []uuid.UUID, error) {
	return getAndUnmarshalBatch(ctx, redis.ScopeUser, c.client, ids, userKey, unmarshalUser)
}

func (c *UserCache) SetBatch(ctx context.Context, users map[uuid.UUID]*user.User) error {
	return marshalAndSetBatch(ctx, redis.ScopeUser, c.client, users, userKey, userTTL, marshalUser)
}

func (c *UserCache) DeleteBatch(ctx context.Context, ids []uuid.UUID) error {
	return deleteBatch(ctx, redis.ScopeUser, c.client, ids, userKey)
}

// -----------------------------------------------------------------------------
// Channel Operations
// -----------------------------------------------------------------------------

// SetChannelIDs replaces the user's cached channels ZSet using ZADD inside a pipeline.
func (c *UserCache) SetChannelIDs(ctx context.Context, userID uuid.UUID, channelIDs []uuid.UUID) error {
	key := userChannelsKey(userID)
	if len(channelIDs) == 0 {
		return c.client.Del(ctx, key).Err()
	}

	zMembers := make([]redisdriver.Z, len(channelIDs))
	for i, chID := range channelIDs {
		zMembers[i] = redisdriver.Z{
			Score:  0,
			Member: chID.String(),
		}
	}

	pipe := c.client.Pipeline()
	pipe.Del(ctx, key)
	pipe.ZAdd(ctx, key, zMembers...)
	pipe.Expire(ctx, key, userChannelsTTL)

	if _, err := pipe.Exec(ctx); err != nil {
		return redis.NewError(err, redis.ScopeUser)
	}
	return nil
}

// GetChannelIDs fetches all channel IDs from the user's channels ZSet using ZRANGE.
func (c *UserCache) GetChannelIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	vals, err := c.client.ZRange(ctx, userChannelsKey(userID), 0, -1).Result()
	if err != nil {
		if errors.Is(err, redisdriver.Nil) {
			return nil, nil
		}
		return nil, redis.NewError(err, redis.ScopeUser)
	}

	ids := make([]uuid.UUID, 0, len(vals))
	for _, val := range vals {
		if id, parseErr := uuid.Parse(val); parseErr == nil {
			ids = append(ids, uuid.UUID(id))
		}
	}

	return ids, nil
}

// RemoveChannelID removes a channel ID from the user's channels ZSet using ZREM.
func (c *UserCache) RemoveChannelID(ctx context.Context, userID, channelID uuid.UUID) error {
	if err := c.client.ZRem(ctx, userChannelsKey(userID), channelID.String()).Err(); err != nil {
		return redis.NewError(err, redis.ScopeUser)
	}
	return nil
}

// AddChannelID adds a channel ID to the user's channels ZSet using ZADD.
func (c *UserCache) AddChannelID(ctx context.Context, userID, channelID uuid.UUID) error {
	pipe := c.client.Pipeline()
	pipe.ZAdd(ctx, userChannelsKey(userID), redisdriver.Z{
		Score:  0,
		Member: channelID.String(),
	})
	pipe.Expire(ctx, userChannelsKey(userID), userChannelsTTL)

	if _, err := pipe.Exec(ctx); err != nil {
		return redis.NewError(err, redis.ScopeUser)
	}
	return nil
}

func (c *UserCache) GetPeerIDs(
	ctx context.Context,
	userID uuid.UUID,
) ([]uuid.UUID, error) {
	userStr := userID.String()

	channelIDs, err := c.GetChannelIDs(ctx, userID)
	if err != nil || len(channelIDs) == 0 {
		return nil, err
	}

	if len(channelIDs) > 100 {
		channelIDs = channelIDs[:100]
	}

	channelCmds := make([]*redisdriver.StringSliceCmd, 0, len(channelIDs))
	_, err = c.client.Pipelined(ctx, func(pipe redisdriver.Pipeliner) error {
		for _, chID := range channelIDs {
			channelCmds = append(channelCmds, pipe.SMembers(ctx, channelMembersKey(chID)))
		}
		return nil
	})
	if err != nil && !errors.Is(err, redisdriver.Nil) {
		return nil, redis.NewError(err, redis.ScopeUser)
	}

	rawMemberCount := 0
	for _, cmd := range channelCmds {
		if members, cmdErr := cmd.Result(); cmdErr == nil {
			rawMemberCount += len(members)
		}
	}
	if rawMemberCount == 0 {
		return nil, nil
	}

	candidateSet := make(map[string]struct{}, min(rawMemberCount, 1000))
	for _, cmd := range channelCmds {
		members, cmdErr := cmd.Result()
		if cmdErr != nil {
			continue
		}
		for _, mStr := range members {
			if mStr != "" && mStr != userStr {
				candidateSet[mStr] = struct{}{}
			}
		}
	}

	candidates := make([]uuid.UUID, 0, len(candidateSet))
	for mStr := range candidateSet {
		if id, parseErr := uuid.Parse(mStr); parseErr == nil {
			candidates = append(candidates, uuid.UUID(id))
		}
	}

	return candidates, nil
}

// GetVisibleMembersByUserID retrieves member representations for the given user across their cached channel ZSet.
func (c *UserCache) GetVisibleMembersByUserID(
	ctx context.Context,
	userID uuid.UUID,
	limit int,
) ([]*channel.Member, bool, error) {
	// 1. Get ordered channel IDs for the user using UserCache's native method
	channelIDs, err := c.GetChannelIDs(ctx, userID)
	if err != nil || len(channelIDs) == 0 {
		return nil, false, err
	}

	if limit > 0 && len(channelIDs) > limit {
		channelIDs = channelIDs[:limit]
	}

	// 2. Fetch the user's member object across all these channel Hash keys in a single pipeline
	pipe := c.client.Pipeline()
	userStr := userID.String()
	cmds := make([]*redisdriver.StringCmd, len(channelIDs))

	for i, chID := range channelIDs {
		cmds[i] = pipe.HGet(ctx, channelMembersKey(chID), userStr)
	}

	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redisdriver.Nil) {
		return nil, false, redis.NewError(err, redis.ScopeChannel)
	}

	members := make([]*channel.Member, 0, len(channelIDs))
	for _, cmd := range cmds {
		rawJSON, err := cmd.Result()
		if err != nil {
			// Missing field or hash -> partial cache miss
			return nil, false, nil
		}

		var mDTO Member
		if err := json.Unmarshal([]byte(rawJSON), &mDTO); err != nil {
			return nil, false, nil
		}

		members = append(members, mDTO.ToDomain())
	}

	return members, true, nil
}
