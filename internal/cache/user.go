package cache

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"bonfire-api/internal/fields"
	"bonfire-api/internal/redis"
	"bonfire-api/internal/user"

	"github.com/google/uuid"
	redisdriver "github.com/redis/go-redis/v9"
)

func userPresenceKey(id fields.ID) string {
	return fmt.Sprintf("{user:%s}:presence", id.String())
}

func userNodesKey(id fields.ID) string {
	return fmt.Sprintf("{user:%s}:nodes", id.String())
}

type UserCache struct {
	client redisdriver.Cmdable
	scope  redis.Scope
	ttl    time.Duration
}

func NewUserCache(client redisdriver.Cmdable, scope redis.Scope, ttl time.Duration) *UserCache {
	return &UserCache{
		client: client,
		scope:  scope,
		ttl:    ttl,
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
	if err := c.client.Set(ctx, userPresenceKey(userID), uint8(p.Int()), c.ttl).Err(); err != nil {
		return redis.NewError(err, c.scope)
	}
	return nil
}

// --- Node Presence Operations ---

func (c *UserCache) AddNode(ctx context.Context, userID fields.ID, nodeID string) error {
	pKey := userPresenceKey(userID)
	nKey := userNodesKey(userID)

	_, err := c.client.Pipelined(ctx, func(pipe redisdriver.Pipeliner) error {
		pipe.SAdd(ctx, nKey, nodeID)
		pipe.Expire(ctx, nKey, c.ttl)
		pipe.Expire(ctx, pKey, c.ttl)
		return nil
	})
	if err != nil {
		return redis.NewError(err, c.scope)
	}
	return nil
}

func (c *UserCache) RemoveNode(ctx context.Context, userID fields.ID, nodeID string) error {
	if err := c.client.SRem(ctx, userNodesKey(userID), nodeID).Err(); err != nil {
		return redis.NewError(err, c.scope)
	}
	return nil
}

func (c *UserCache) GetNodesForUsers(ctx context.Context, userIDs []fields.ID) (map[string][]uuid.UUID, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}

	cmds := make(map[fields.ID]*redisdriver.StringSliceCmd, len(userIDs))
	_, err := c.client.Pipelined(ctx, func(pipe redisdriver.Pipeliner) error {
		for _, id := range userIDs {
			cmds[id] = pipe.SMembers(ctx, userNodesKey(id))
		}
		return nil
	})
	if err != nil && err != redisdriver.Nil {
		return nil, redis.NewError(err, c.scope)
	}

	nodeToUsers := make(map[string][]uuid.UUID)
	for id, cmd := range cmds {
		for _, nodeID := range cmd.Val() {
			if nodeID != "" {
				nodeToUsers[nodeID] = append(nodeToUsers[nodeID], id.UUID())
			}
		}
	}

	return nodeToUsers, nil
}

func (c *UserCache) Heartbeat(ctx context.Context, userID fields.ID) error {
	pKey := userPresenceKey(userID)
	nKey := userNodesKey(userID)

	_, err := c.client.Pipelined(ctx, func(pipe redisdriver.Pipeliner) error {
		pipe.Expire(ctx, pKey, c.ttl)
		pipe.Expire(ctx, nKey, c.ttl)
		return nil
	})
	if err != nil {
		return redis.NewError(err, c.scope)
	}

	return nil
}
