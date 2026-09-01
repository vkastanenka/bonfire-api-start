package cache

import (
	"context"
	"strconv"
	"time"

	"bonfire-api/internal/fields"
	"bonfire-api/internal/redis"
	"bonfire-api/internal/user"

	"github.com/google/uuid"
	redisdriver "github.com/redis/go-redis/v9"
)

var (
	userPresenceTTL = 45 * time.Second
	userNodesTTL    = 45 * time.Second
	userFriendsTTL  = 24 * time.Hour
	userChannelsTTL = 24 * time.Hour
)

const (
	userDomainKey = "user:"
)

func userChannelsKey(id fields.ID) string {
	return "{" + userDomainKey + id.String() + "}:channels"
}

func userFriendsKey(id fields.ID) string {
	return "{" + userDomainKey + id.String() + "}:friends"
}

func userNodesKey(id fields.ID) string {
	return "{" + userDomainKey + id.String() + "}:nodes"
}

func userPresenceKey(id fields.ID) string {
	return "{" + userDomainKey + id.String() + "}:presence"
}

type UserCache struct {
	client redisdriver.Cmdable
	scope  redis.Scope
}

func NewUserCache(client redisdriver.Cmdable, scope redis.Scope) *UserCache {
	return &UserCache{
		client: client,
		scope:  scope,
	}
}

// --- Presence Operations ---

func (c *UserCache) GetPresence(ctx context.Context, userID fields.ID) (user.Presence, error) {
	val, err := c.client.Get(ctx, userPresenceKey(userID)).Uint64()
	if redis.IsCacheMiss(err) {
		return user.NewPresenceOffline(), nil
	}
	if err != nil {
		return user.NewPresenceOffline(), redis.NewError(err, c.scope)
	}

	p, err := user.ParsePresence(int(val))
	if err != nil {
		return user.NewPresenceOffline(), nil
	}

	return p, nil
}

func (c *UserCache) GetBatchPresence(
	ctx context.Context,
	userIDs []fields.ID,
) (map[fields.ID]user.Presence, error) {
	result := make(map[fields.ID]user.Presence, len(userIDs))
	if len(userIDs) == 0 {
		return result, nil
	}

	cmds := make(map[fields.ID]*redisdriver.StringCmd, len(userIDs))
	_, err := c.client.Pipelined(ctx, func(pipe redisdriver.Pipeliner) error {
		for _, id := range userIDs {
			cmds[id] = pipe.Get(ctx, userPresenceKey(id))
		}
		return nil
	})
	if err != nil && err != redisdriver.Nil {
		return nil, redis.NewError(err, c.scope)
	}

	for id, cmd := range cmds {
		valStr, parseErr := cmd.Result()
		if parseErr != nil {
			result[id] = user.NewPresenceOffline()
			continue
		}

		val, parseErr := strconv.ParseUint(valStr, 10, 8)
		if parseErr != nil {
			result[id] = user.NewPresenceOffline()
			continue
		}

		p, parseErr := user.ParsePresence(int(val))
		if parseErr != nil {
			result[id] = user.NewPresenceOffline()
			continue
		}

		result[id] = p
	}

	return result, nil
}

func (c *UserCache) SetPresence(ctx context.Context, userID fields.ID, p user.Presence) error {
	if err := c.client.Set(ctx, userPresenceKey(userID), uint8(p.Int()), userPresenceTTL).Err(); err != nil {
		return redis.NewError(err, c.scope)
	}
	return nil
}

// --- Node Operations ---

func (c *UserCache) AddNode(ctx context.Context, userID, nodeID fields.ID) error {
	pKey := userPresenceKey(userID)
	nKey := userNodesKey(userID)

	_, err := c.client.Pipelined(ctx, func(pipe redisdriver.Pipeliner) error {
		pipe.SAdd(ctx, nKey, nodeID.String())
		pipe.Expire(ctx, nKey, userNodesTTL)
		pipe.Expire(ctx, pKey, userPresenceTTL)
		return nil
	})
	if err != nil {
		return redis.NewError(err, c.scope)
	}
	return nil
}

func (c *UserCache) RemoveNode(ctx context.Context, userID, nodeID fields.ID) error {
	if err := c.client.SRem(ctx, userNodesKey(userID), nodeID).Err(); err != nil {
		return redis.NewError(err, c.scope)
	}
	return nil
}

func (c *UserCache) RemoveBatchNode(ctx context.Context, userIDs []fields.ID, nodeID fields.ID) error {
	if len(userIDs) == 0 {
		return nil
	}

	_, err := c.client.Pipelined(ctx, func(pipe redisdriver.Pipeliner) error {
		for _, userID := range userIDs {
			pipe.SRem(ctx, userNodesKey(userID), nodeID.String())
		}
		return nil
	})
	if err != nil {
		return redis.NewError(err, c.scope)
	}

	return nil
}

// Using ZSET: Score is current epoch timestamp. Inactive nodes are removed via ZREMRANGEBYSCORE.
func (c *UserCache) Heartbeat(ctx context.Context, userID fields.ID, nodeID fields.ID) error {
	nKey := userNodesKey(userID)
	pKey := userPresenceKey(userID)
	now := time.Now().Unix()

	_, err := c.client.Pipelined(ctx, func(pipe redisdriver.Pipeliner) error {
		pipe.ZAdd(ctx, nKey, redisdriver.Z{Score: float64(now), Member: nodeID.String()})
		pipe.Expire(ctx, nKey, userNodesTTL)
		pipe.Expire(ctx, pKey, userPresenceTTL)
		return nil
	})
	return err
}

func (c *UserCache) RegisterWSConnection(ctx context.Context, userID, nodeID fields.ID, presence user.Presence) error {
	pKey := userPresenceKey(userID)
	nKey := userNodesKey(userID)

	_, err := c.client.Pipelined(ctx, func(pipe redisdriver.Pipeliner) error {
		pipe.SAdd(ctx, nKey, nodeID.String())
		pipe.Expire(ctx, nKey, userNodesTTL)
		pipe.Set(ctx, pKey, presence.Int(), userPresenceTTL)
		return nil
	})
	if err != nil {
		return redis.NewError(err, c.scope)
	}

	return nil
}

var unregisterScript = redisdriver.NewScript(`
	redis.call("SREM", KEYS[1], ARGV[1])
	local count = redis.call("SCARD", KEYS[1])
	if count == 0 then
		redis.call("SET", KEYS[2], ARGV[2], "EX", ARGV[3])
		return 1 -- User went completely offline
	end
	return 0 -- User still has active nodes
`)

func (c *UserCache) UnregisterWSConnection(ctx context.Context, userID, nodeID fields.ID) (bool, error) {
	nKey := userNodesKey(userID)
	pKey := userPresenceKey(userID)

	res, err := unregisterScript.Run(
		ctx,
		c.client,
		[]string{nKey, pKey},
		nodeID.String(),
		user.NewPresenceOffline().Int(),
		int(userPresenceTTL.Seconds()),
	).Int()

	if err != nil {
		return false, redis.NewError(err, c.scope)
	}

	return res == 1, nil
}

// --- Friend Operations ---

func (c *UserCache) AddFriend(ctx context.Context, userID, friendID fields.ID) error {
	fKey := userFriendsKey(userID)

	_, err := c.client.Pipelined(ctx, func(pipe redisdriver.Pipeliner) error {
		pipe.SAdd(ctx, fKey, friendID.String())
		pipe.Expire(ctx, fKey, userFriendsTTL)
		return nil
	})
	if err != nil {
		return redis.NewError(err, c.scope)
	}

	return nil
}

func (c *UserCache) RemoveFriend(ctx context.Context, userID, friendID fields.ID) error {
	fKey := userFriendsKey(userID)

	if err := c.client.SRem(ctx, fKey, friendID.String()).Err(); err != nil {
		return redis.NewError(err, c.scope)
	}

	return nil
}

// SetFriends populates or replaces the friend list cache for a user with a given TTL
func (c *UserCache) SetFriends(ctx context.Context, userID fields.ID, friendIDs []fields.ID, ttl time.Duration) error {
	fKey := userFriendsKey(userID)

	_, err := c.client.Pipelined(ctx, func(pipe redisdriver.Pipeliner) error {
		pipe.Del(ctx, fKey) // Clear existing cache first if overwriting
		if len(friendIDs) > 0 {
			members := make([]interface{}, len(friendIDs))
			for i, id := range friendIDs {
				members[i] = id.String()
			}
			pipe.SAdd(ctx, fKey, members...)
			pipe.Expire(ctx, fKey, ttl)
		}
		return nil
	})
	if err != nil {
		return redis.NewError(err, c.scope)
	}
	return nil
}

// GetFriends retrieves all cached friend IDs for a user
func (c *UserCache) GetFriends(ctx context.Context, userID fields.ID) ([]fields.ID, error) {
	fKey := userFriendsKey(userID)

	members, err := c.client.SMembers(ctx, fKey).Result()
	if redis.IsCacheMiss(err) {
		return nil, nil
	}
	if err != nil {
		return nil, redis.NewError(err, c.scope)
	}

	friendIDs := make([]fields.ID, 0, len(members))
	for _, m := range members {
		id, parseErr := uuid.Parse(m)
		if parseErr == nil {
			friendIDs = append(friendIDs, fields.ID(id))
		}
	}

	return friendIDs, nil
}

// --- Channel Operations ---

func (c *UserCache) AddChannel(ctx context.Context, userID, channelID fields.ID) error {
	chKey := userChannelsKey(userID)

	_, err := c.client.Pipelined(ctx, func(pipe redisdriver.Pipeliner) error {
		pipe.SAdd(ctx, chKey, channelID.String())
		pipe.Expire(ctx, chKey, userChannelsTTL)
		return nil
	})
	if err != nil {
		return redis.NewError(err, c.scope)
	}
	return nil
}

func (c *UserCache) RemoveChannel(ctx context.Context, userID, channelID fields.ID) error {
	chKey := userChannelsKey(userID)

	if err := c.client.SRem(ctx, chKey, channelID.String()).Err(); err != nil {
		return redis.NewError(err, c.scope)
	}
	return nil
}

func (c *UserCache) SetChannels(ctx context.Context, userID fields.ID, channelIDs []fields.ID, ttl time.Duration) error {
	chKey := userChannelsKey(userID)

	_, err := c.client.Pipelined(ctx, func(pipe redisdriver.Pipeliner) error {
		pipe.Del(ctx, chKey)
		if len(channelIDs) > 0 {
			members := make([]interface{}, len(channelIDs))
			for i, id := range channelIDs {
				members[i] = id.String()
			}
			pipe.SAdd(ctx, chKey, members...)
			pipe.Expire(ctx, chKey, ttl)
		}
		return nil
	})
	if err != nil {
		return redis.NewError(err, c.scope)
	}
	return nil
}

// GetBroadcastRecipients fetches deduplicated recipient user IDs across
// a user's cached friends list and active channel members in 2 RTTs.
func (c *UserCache) GetBroadcastRecipients(
	ctx context.Context,
	userID fields.ID,
) ([]fields.ID, error) {
	friendsKey := userFriendsKey(userID)
	channelsKey := userChannelsKey(userID)

	var (
		friendsCmd  *redisdriver.StringSliceCmd
		channelsCmd *redisdriver.StringSliceCmd
	)

	// --- RTT 1: Fetch friends and joined channels in a single pipeline ---
	_, err := c.client.Pipelined(ctx, func(pipe redisdriver.Pipeliner) error {
		friendsCmd = pipe.SMembers(ctx, friendsKey)
		channelsCmd = pipe.SMembers(ctx, channelsKey)
		return nil
	})
	if err != nil && err != redisdriver.Nil {
		return nil, redis.NewError(err, c.scope)
	}

	friendStrs, _ := friendsCmd.Result()
	channelStrs, _ := channelsCmd.Result()

	if len(friendStrs) == 0 && len(channelStrs) == 0 {
		return nil, nil
	}

	// --- RTT 2: Fan out channel queries concurrently across Redis cluster nodes ---
	channelCmds := make([]*redisdriver.StringSliceCmd, 0, len(channelStrs))
	if len(channelStrs) > 0 {
		_, err = c.client.Pipelined(ctx, func(pipe redisdriver.Pipeliner) error {
			for _, chStr := range channelStrs {
				chUUID, parseErr := uuid.Parse(chStr)
				if parseErr != nil {
					continue
				}
				actKey := channelActiveUsersKey(fields.ID(chUUID))
				channelCmds = append(channelCmds, pipe.SMembers(ctx, actKey))
			}
			return nil
		})
		if err != nil && err != redisdriver.Nil {
			return nil, redis.NewError(err, c.scope)
		}
	}

	// Pre-size map based on estimated unique recipients to avoid map resizings
	totalEstimated := len(friendStrs)
	for _, cmd := range channelCmds {
		if members, _ := cmd.Result(); len(members) > 0 {
			totalEstimated += len(members)
		}
	}

	recipientMap := make(map[fields.ID]struct{}, totalEstimated)

	// Parse friends directly into fields.ID
	for _, fStr := range friendStrs {
		if id, parseErr := uuid.Parse(fStr); parseErr == nil && id != uuid.UUID(userID) {
			recipientMap[fields.ID(id)] = struct{}{}
		}
	}

	// Parse channel members directly into fields.ID
	for _, cmd := range channelCmds {
		members, parseErr := cmd.Result()
		if parseErr != nil {
			continue
		}
		for _, mStr := range members {
			if id, parseErr := uuid.Parse(mStr); parseErr == nil && id != uuid.UUID(userID) {
				recipientMap[fields.ID(id)] = struct{}{}
			}
		}
	}

	// Extract keys directly without re-parsing
	recipients := make([]fields.ID, 0, len(recipientMap))
	for id := range recipientMap {
		recipients = append(recipients, id)
	}

	return recipients, nil
}
