package cache

import (
	"context"
	"errors"
	"time"

	"bonfire-api/internal/fields"
	"bonfire-api/internal/redis"
	"bonfire-api/internal/user"

	"github.com/google/uuid"
	redisdriver "github.com/redis/go-redis/v9"
)

var (
	userTTL         = 24 * time.Hour
	userPresenceTTL = 90 * time.Second
	userNodesTTL    = 90 * time.Second
	userFriendsTTL  = 24 * time.Hour
	userChannelsTTL = 24 * time.Hour
)

const (
	userDomainKey = "user:"
)

func userNamespacedKey(id fields.ID, suffix string) string {
	if suffix == "" {
		return "{" + userDomainKey + id.String() + "}"
	}
	return "{" + userDomainKey + id.String() + "}:" + suffix
}

func userKey(id fields.ID) string               { return userNamespacedKey(id, "") }
func userPresenceKey(id fields.ID) string       { return userNamespacedKey(id, "presence") }
func userSessionsKey(id fields.ID) string       { return userNamespacedKey(id, "sessions") }
func userNodesKey(id fields.ID) string          { return userNamespacedKey(id, "nodes") }
func userFriendsKey(id fields.ID) string        { return userNamespacedKey(id, "friends") }
func userFriendRequestsKey(id fields.ID) string { return userNamespacedKey(id, "friend_requests") }
func userBlocksKey(id fields.ID) string         { return userNamespacedKey(id, "blocks") }
func userChannelsKey(id fields.ID) string       { return userNamespacedKey(id, "channels") }

type UserCache struct {
	client redisdriver.Cmdable
}

func NewUserCache(client redisdriver.Cmdable) *UserCache {
	return &UserCache{
		client: client,
	}
}

func (c *UserCache) Get(ctx context.Context, id fields.ID) (*user.User, error) {
	return getAndUnmarshal(ctx, c.client, userKey(id), redis.ScopeUser, unmarshalUser)
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

func (c *UserCache) Set(ctx context.Context, usr *user.User) error {
	return marshalAndSet(ctx, c.client, userKey(usr.ID()), usr, userTTL, redis.ScopeUser, marshalUser)
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

func (c *UserCache) Delete(ctx context.Context, id fields.ID) error {
	if err := c.client.Del(ctx, userKey(id)).Err(); err != nil {
		return redis.NewError(err, redis.ScopeUser)
	}
	return nil
}

func (c *UserCache) DeleteBatch(ctx context.Context, ids []fields.ID) error {
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = userKey(id)
	}
	return deleteBatchKeys(ctx, c.client, keys, redis.ScopeUser)
}

func (c *UserCache) SetFriendIDs(ctx context.Context, userID fields.ID, friendIDs []fields.ID) error {
	return setSetIDs(ctx, c.client, userFriendsKey(userID), friendIDs, userFriendsTTL, redis.ScopeUser)
}

func (c *UserCache) GetFriendIDs(ctx context.Context, userID fields.ID) ([]fields.ID, error) {
	return getSetIDs(ctx, c.client, userFriendsKey(userID), redis.ScopeUser)
}

func (c *UserCache) AddFriendID(ctx context.Context, userID, friendID fields.ID) error {
	return addToSetID(ctx, c.client, userFriendsKey(userID), friendID, userFriendsTTL, redis.ScopeUser)
}

func (c *UserCache) RemoveFriendID(ctx context.Context, userID, friendID fields.ID) error {
	return removeFromSetID(ctx, c.client, userFriendsKey(userID), friendID, redis.ScopeUser)
}

func (c *UserCache) SetChannelIDs(ctx context.Context, userID fields.ID, channelIDs []fields.ID) error {
	return setSetIDs(ctx, c.client, userChannelsKey(userID), channelIDs, userChannelsTTL, redis.ScopeUser)
}

func (c *UserCache) GetChannelIDs(ctx context.Context, userID fields.ID) ([]fields.ID, error) {
	return getSetIDs(ctx, c.client, userChannelsKey(userID), redis.ScopeUser)
}

func (c *UserCache) AddChannelID(ctx context.Context, userID, channelID fields.ID) error {
	return addToSetID(ctx, c.client, userChannelsKey(userID), channelID, userChannelsTTL, redis.ScopeUser)
}

func (c *UserCache) RemoveChannelID(ctx context.Context, userID, channelID fields.ID) error {
	return removeFromSetID(ctx, c.client, userChannelsKey(userID), channelID, redis.ScopeUser)
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
