package cache

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"bonfire-api/internal/channel"
	"bonfire-api/internal/fields"
	"bonfire-api/internal/redis"
	"bonfire-api/internal/user"

	"github.com/google/uuid"
	redisdriver "github.com/redis/go-redis/v9"
)

var (
	userTTL         = 24 * time.Hour
	userFriendsTTL  = 24 * time.Hour
	userBlocksTTL   = 24 * time.Hour
	userChannelsTTL = 24 * time.Hour
)

const (
	emptySetSentinel = "__empty__"
)

func userKey(id fields.ID) string          { return "{user:" + id.String() + "}" }
func userPresenceKey(id fields.ID) string  { return "{user:" + id.String() + "}:presence" }
func userSessionsKey(id fields.ID) string  { return "{user:" + id.String() + "}:sessions" }
func userFriendsKey(id fields.ID) string   { return "{user:" + id.String() + "}:friends" }
func userBlocksKey(id fields.ID) string    { return "{user:" + id.String() + "}:blocks" }
func userBlockedByKey(id fields.ID) string { return "{user:" + id.String() + "}:blocked_by" }
func userChannelsKey(id fields.ID) string  { return "{user:" + id.String() + "}:channels" }

type UserCache struct {
	client redisdriver.Cmdable
}

func NewUserCache(client redisdriver.Cmdable) *UserCache {
	return &UserCache{
		client: client,
	}
}

// -----------------------------------------------------------------------------
// User Profile Operations
// -----------------------------------------------------------------------------

func (c *UserCache) Get(ctx context.Context, id fields.ID) (*user.User, error) {
	return getAndUnmarshal(ctx, c.client, userKey(id), redis.ScopeUser, unmarshalUser)
}

func (c *UserCache) Set(ctx context.Context, usr *user.User) error {
	return marshalAndSet(ctx, c.client, userKey(usr.ID()), usr, userTTL, redis.ScopeUser, marshalUser)
}

func (c *UserCache) Delete(ctx context.Context, id fields.ID) error {
	if err := c.client.Del(ctx, userKey(id)).Err(); err != nil {
		return redis.NewError(err, redis.ScopeUser)
	}
	return nil
}

func (c *UserCache) GetBatch(
	ctx context.Context,
	ids []fields.ID,
) (map[fields.ID]*user.User, []fields.ID, error) {
	if len(ids) == 0 {
		return make(map[fields.ID]*user.User), nil, nil
	}

	found := make(map[fields.ID]*user.User, len(ids))
	missing := make([]fields.ID, 0, len(ids))
	var corruptedKeys []string

	for i := 0; i < len(ids); i += MaxBatchSize {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}

		end := min(i+MaxBatchSize, len(ids))
		chunk := ids[i:end]

		redisKeys := make([]string, len(chunk))
		for j, id := range chunk {
			redisKeys[j] = userKey(id)
		}

		vals, err := getBatchKeys(ctx, c.client, redisKeys, redis.ScopeUser)
		if err != nil {
			return nil, nil, err
		}

		for j, raw := range vals {
			id := chunk[j]
			rKey := redisKeys[j]

			data, ok := toBytes(raw)
			if !ok {
				missing = append(missing, id)
				continue
			}

			usr, err := unmarshalUser(data)
			if err != nil {
				corruptedKeys = append(corruptedKeys, rKey)
				missing = append(missing, id)
				continue
			}

			found[id] = usr
		}
	}

	if len(corruptedKeys) > 0 {
		deleteBatchKeys(ctx, c.client, corruptedKeys, redis.ScopeUser)
	}

	return found, missing, nil
}

func (c *UserCache) SetBatch(ctx context.Context, users map[fields.ID]*user.User) error {
	if len(users) == 0 {
		return nil
	}

	items := make([]CacheItem, 0, len(users))
	for id, usr := range users {
		if usr == nil || id.IsZero() {
			continue
		}

		bytes, err := marshalUser(usr)
		if err != nil {
			return err
		}

		items = append(items, CacheItem{
			Key:   userKey(id),
			Value: bytes,
		})
	}

	return setBatchPipeline(ctx, c.client, items, userTTL, redis.ScopeUser)
}

func (c *UserCache) DeleteBatch(ctx context.Context, ids []fields.ID) error {
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = userKey(id)
	}
	return deleteBatchKeys(ctx, c.client, keys, redis.ScopeUser)
}

// -----------------------------------------------------------------------------
// Friends Operations
// -----------------------------------------------------------------------------

// GetFriendIDs fetches a user's friend IDs. Returns (nil, nil) on cache miss.
func (c *UserCache) GetFriendIDs(ctx context.Context, userID fields.ID) ([]fields.ID, error) {
	return c.getRelationIDs(ctx, userFriendsKey(userID))
}

// SetFriendIDs sets the full list of friend IDs, caching a sentinel if the slice is empty.
func (c *UserCache) SetFriendIDs(ctx context.Context, userID fields.ID, friendIDs []fields.ID) error {
	return c.setRelationIDs(ctx, userFriendsKey(userID), friendIDs, userFriendsTTL)
}

// AddFriendID adds a single friend to a user's friend set.
func (c *UserCache) AddFriendID(ctx context.Context, userID, friendID fields.ID) error {
	return c.addRelationID(ctx, userFriendsKey(userID), friendID, userFriendsTTL)
}

// RemoveFriendPair atomically removes two users from each other's friend sets.
func (c *UserCache) RemoveFriendPair(ctx context.Context, userA, userB fields.ID) error {
	removals := map[string]fields.ID{
		userFriendsKey(userA): userB,
		userFriendsKey(userB): userA,
	}
	return removeFromSetIDsPipelined(ctx, c.client, removals, redis.ScopeUser)
}

// -----------------------------------------------------------------------------
// Block Operations
// -----------------------------------------------------------------------------

// GetBlocklistIDs retrieves all user IDs that the given user has blocked.
func (c *UserCache) GetBlocklistIDs(ctx context.Context, userID fields.ID) ([]fields.ID, error) {
	return c.getRelationIDs(ctx, userBlocksKey(userID))
}

// SetBlocklistIDs sets the complete list of user IDs blocked by this user.
func (c *UserCache) SetBlocklistIDs(ctx context.Context, userID fields.ID, blockedIDs []fields.ID) error {
	return c.setRelationIDs(ctx, userBlocksKey(userID), blockedIDs, userBlocksTTL)
}

// GetBlockedByIDs retrieves all user IDs that have blocked the given user.
func (c *UserCache) GetBlockedByIDs(ctx context.Context, userID fields.ID) ([]fields.ID, error) {
	return c.getRelationIDs(ctx, userBlockedByKey(userID))
}

// SetBlockedByIDs sets the complete list of user IDs who have blocked this user.
func (c *UserCache) SetBlockedByIDs(ctx context.Context, userID fields.ID, blockerIDs []fields.ID) error {
	return c.setRelationIDs(ctx, userBlockedByKey(userID), blockerIDs, userBlocksTTL)
}

// BlockUser performs an atomic bi-directional block update:
// 1. Adds targetID to blockerID's blocklist ({user:blocker}:blocks).
// 2. Adds blockerID to targetID's blocked_by list ({user:target}:blocked_by).
// 3. Removes both users from each other's friend sets if a friendship existed.
func (c *UserCache) BlockUser(ctx context.Context, blockerID, targetID fields.ID) error {
	pipe := c.client.Pipeline()

	// 1. Update Block Sets
	pipe.SAdd(ctx, userBlocksKey(blockerID), targetID.String())
	pipe.ExpireXX(ctx, userBlocksKey(blockerID), userBlocksTTL)
	pipe.SRem(ctx, userBlocksKey(blockerID), emptySetSentinel)

	pipe.SAdd(ctx, userBlockedByKey(targetID), blockerID.String())
	pipe.ExpireXX(ctx, userBlockedByKey(targetID), userBlocksTTL)
	pipe.SRem(ctx, userBlockedByKey(targetID), emptySetSentinel)

	// 2. Sever Friendship if cached
	pipe.SRem(ctx, userFriendsKey(blockerID), targetID.String())
	pipe.SRem(ctx, userFriendsKey(targetID), blockerID.String())

	if _, err := pipe.Exec(ctx); err != nil {
		return redis.NewError(err, redis.ScopeUser)
	}
	return nil
}

// UnblockUser removes targetID from blockerID's blocklist and blockerID from targetID's blocked_by set.
func (c *UserCache) UnblockUser(ctx context.Context, blockerID, targetID fields.ID) error {
	removals := map[string]fields.ID{
		userBlocksKey(blockerID):   targetID,
		userBlockedByKey(targetID): blockerID,
	}
	return removeFromSetIDsPipelined(ctx, c.client, removals, redis.ScopeUser)
}

// -----------------------------------------------------------------------------
// Shared Set Abstractions (Handles Sentinel Filtering & Cache Penetration)
// -----------------------------------------------------------------------------

func (c *UserCache) getRelationIDs(ctx context.Context, key string) ([]fields.ID, error) {
	members, err := c.client.SMembers(ctx, key).Result()
	if errors.Is(err, redisdriver.Nil) || len(members) == 0 {
		return nil, nil // Cache miss
	}
	if err != nil {
		return nil, redis.NewError(err, redis.ScopeUser)
	}

	ids := make([]fields.ID, 0, len(members))
	for _, m := range members {
		if m == emptySetSentinel {
			continue
		}
		if id, parseErr := uuid.Parse(m); parseErr == nil {
			ids = append(ids, fields.ID(id))
		}
	}

	return ids, nil
}

func (c *UserCache) setRelationIDs(ctx context.Context, key string, ids []fields.ID, ttl time.Duration) error {
	_, err := c.client.TxPipelined(ctx, func(pipe redisdriver.Pipeliner) error {
		pipe.Del(ctx, key)
		if len(ids) == 0 {
			pipe.SAdd(ctx, key, emptySetSentinel)
		} else {
			members := make([]interface{}, len(ids))
			for i, id := range ids {
				members[i] = id.String()
			}
			pipe.SAdd(ctx, key, members...)
		}
		pipe.Expire(ctx, key, ttl)
		return nil
	})

	if err != nil {
		return redis.NewError(err, redis.ScopeUser)
	}
	return nil
}

func (c *UserCache) addRelationID(ctx context.Context, key string, targetID fields.ID, ttl time.Duration) error {
	_, err := c.client.TxPipelined(ctx, func(pipe redisdriver.Pipeliner) error {
		pipe.SAdd(ctx, key, targetID.String())
		pipe.SRem(ctx, key, emptySetSentinel)
		pipe.ExpireXX(ctx, key, ttl)
		return nil
	})
	if err != nil {
		return redis.NewError(err, redis.ScopeUser)
	}
	return nil
}

// -----------------------------------------------------------------------------
// Channel Operations
// -----------------------------------------------------------------------------

// SetChannelIDs replaces the user's cached channels ZSet using ZADD inside a pipeline.
func (c *UserCache) SetChannelIDs(ctx context.Context, userID fields.ID, channelIDs []fields.ID) error {
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
func (c *UserCache) GetChannelIDs(ctx context.Context, userID fields.ID) ([]fields.ID, error) {
	vals, err := c.client.ZRange(ctx, userChannelsKey(userID), 0, -1).Result()
	if err != nil {
		if errors.Is(err, redisdriver.Nil) {
			return nil, nil
		}
		return nil, redis.NewError(err, redis.ScopeUser)
	}

	ids := make([]fields.ID, 0, len(vals))
	for _, val := range vals {
		if id, parseErr := uuid.Parse(val); parseErr == nil {
			ids = append(ids, fields.ID(id))
		}
	}

	return ids, nil
}

// RemoveChannelID removes a channel ID from the user's channels ZSet using ZREM.
func (c *UserCache) RemoveChannelID(ctx context.Context, userID, channelID fields.ID) error {
	if err := c.client.ZRem(ctx, userChannelsKey(userID), channelID.String()).Err(); err != nil {
		return redis.NewError(err, redis.ScopeUser)
	}
	return nil
}

// AddChannelID adds a channel ID to the user's channels ZSet using ZADD.
func (c *UserCache) AddChannelID(ctx context.Context, userID, channelID fields.ID) error {
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
	userID fields.ID,
) ([]fields.ID, error) {
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

	candidates := make([]fields.ID, 0, len(candidateSet))
	for mStr := range candidateSet {
		if id, parseErr := uuid.Parse(mStr); parseErr == nil {
			candidates = append(candidates, fields.ID(id))
		}
	}

	return candidates, nil
}

// GetVisibleMembersByUserID retrieves member representations for the given user across their cached channel ZSet.
func (c *UserCache) GetVisibleMembersByUserID(
	ctx context.Context,
	userID fields.ID,
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

		m, err := mDTO.ToDomain()
		if err != nil {
			return nil, false, nil
		}

		members = append(members, m)
	}

	return members, true, nil
}
