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

const (
	userDomainKey = "user:"
)

func userNodesKey(id fields.ID) string {
	return userDomainKey + id.String() + ":nodes"
}

func userPresenceKey(id fields.ID) string {
	return userDomainKey + id.String() + ":presence"
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

	redisKeys := make([]string, len(userIDs))
	for j, id := range userIDs {
		redisKeys[j] = userPresenceKey(id)
	}

	vals, err := c.client.MGet(ctx, redisKeys...).Result()
	if err != nil {
		return nil, redis.NewError(err, c.scope)
	}

	for j, raw := range vals {
		userID := userIDs[j]

		valStr, ok := raw.(string)
		if !ok || raw == nil {
			result[userID] = user.NewPresenceOffline()
			continue
		}

		val, parseErr := strconv.ParseUint(valStr, 10, 8)
		if parseErr != nil {
			result[userID] = user.NewPresenceOffline()
			continue
		}

		p, parseErr := user.ParsePresence(int(val))
		if parseErr != nil {
			result[userID] = user.NewPresenceOffline()
			continue
		}

		result[userID] = p
	}

	return result, nil
}

func (c *UserCache) SetPresence(ctx context.Context, userID fields.ID, p user.Presence) error {
	if err := c.client.Set(ctx, userPresenceKey(userID), uint8(p.Int()), c.ttl).Err(); err != nil {
		return redis.NewError(err, c.scope)
	}
	return nil
}

func (c *UserCache) SetBatchPresence(ctx context.Context, items map[fields.ID]user.Presence) error {
	if len(items) == 0 {
		return nil
	}

	pipe := c.client.Pipeline()
	for userID, p := range items {
		pipe.Set(ctx, userPresenceKey(userID), uint8(p.Int()), c.ttl)
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return redis.NewError(err, c.scope)
	}

	return nil
}

// --- Node Presence Operations ---

// AddNode registers a gateway node ID in the user's active node set.
func (c *UserCache) AddNode(ctx context.Context, userID fields.ID, nodeID string) error {
	if err := c.client.SAdd(ctx, userNodesKey(userID), nodeID).Err(); err != nil {
		return redis.NewError(err, c.scope)
	}
	return nil
}

// RemoveNode unregisters a gateway node ID from the user's active node set.
func (c *UserCache) RemoveNode(ctx context.Context, userID fields.ID, nodeID string) error {
	if err := c.client.SRem(ctx, userNodesKey(userID), nodeID).Err(); err != nil {
		return redis.NewError(err, c.scope)
	}
	return nil
}

// RemoveNodeBatch removes this node from multiple users' active node sets in a single pipeline.
func (c *UserCache) RemoveNodeBatch(ctx context.Context, userIDs []fields.ID, nodeID string) error {
	if len(userIDs) == 0 {
		return nil
	}

	pipe := c.client.Pipeline()
	for _, userID := range userIDs {
		pipe.SRem(ctx, userNodesKey(userID), nodeID)
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return redis.NewError(err, c.scope)
	}
	return nil
}

// GetNodes retrieves all active gateway node IDs connected to by the given user.
func (c *UserCache) GetNodes(ctx context.Context, userID fields.ID) ([]string, error) {
	nodes, err := c.client.SMembers(ctx, userNodesKey(userID)).Result()
	if err != nil {
		return nil, redis.NewError(err, c.scope)
	}
	return nodes, nil
}

// ClearNodes removes all node registrations for a user (e.g. forced disconnect or cleanup).
func (c *UserCache) ClearNodes(ctx context.Context, userID fields.ID) error {
	if err := c.client.Del(ctx, userNodesKey(userID)).Err(); err != nil {
		return redis.NewError(err, c.scope)
	}
	return nil
}
