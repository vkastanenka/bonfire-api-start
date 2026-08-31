package cache

import (
	"context"
	"strconv"
	"time"

	"bonfire-api/internal/fields"
	"bonfire-api/internal/redis"
	"bonfire-api/internal/user"

	redisdriver "github.com/redis/go-redis/v9"
)

var (
	userPresenceTTL = 45 * time.Second
	userNodesTTL    = 45 * time.Second
)

const (
	userDomainKey = "user:"
)

func userPresenceKey(id fields.ID) string {
	return "{" + userDomainKey + id.String() + "}:presence"
}

func userNodesKey(id fields.ID) string {
	return "{" + userDomainKey + id.String() + "}:nodes"
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
