package cache

import (
	"context"
	"strconv"

	"bonfire-api/internal/fields"
	"bonfire-api/internal/redis"
	"bonfire-api/internal/user"

	"github.com/google/uuid"
	redisdriver "github.com/redis/go-redis/v9"
)

type PresenceCache struct {
	client redisdriver.Cmdable
}

func NewPresenceCache(client redisdriver.Cmdable) *PresenceCache {
	return &PresenceCache{
		client: client,
	}
}

func (c *PresenceCache) GetPresence(ctx context.Context, userID fields.ID) (user.Presence, error) {
	val, err := c.client.Get(ctx, userPresenceKey(userID)).Result()
	if redis.IsCacheMiss(err) {
		return user.NewPresenceOffline(), nil
	}
	if err != nil {
		return user.NewPresenceOffline(), redis.NewError(err, redis.ScopePresence)
	}

	return parsePresenceString(val), nil
}

func (c *PresenceCache) GetBatchPresence(
	ctx context.Context,
	userIDs []fields.ID,
) (map[fields.ID]user.Presence, error) {
	result := make(map[fields.ID]user.Presence, len(userIDs))
	if len(userIDs) == 0 {
		return result, nil
	}

	for i := 0; i < len(userIDs); i += MaxBatchSize {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		end := min(i+MaxBatchSize, len(userIDs))
		chunk := userIDs[i:end]

		redisKeys := make([]string, len(chunk))
		for j, id := range chunk {
			redisKeys[j] = userPresenceKey(id)
		}

		vals, err := c.client.MGet(ctx, redisKeys...).Result()
		if err != nil {
			return nil, redis.NewError(err, redis.ScopePresence)
		}

		for j, raw := range vals {
			id := chunk[j]

			data, ok := extractBytes(raw)
			if !ok {
				result[id] = user.NewPresenceOffline()
				continue
			}

			result[id] = parsePresenceString(string(data))
		}
	}

	return result, nil
}

func (c *PresenceCache) SetPresence(ctx context.Context, userID fields.ID, p user.Presence) error {
	if err := c.client.Set(ctx, userPresenceKey(userID), uint8(p.Int()), userPresenceTTL).Err(); err != nil {
		return redis.NewError(err, redis.ScopePresence)
	}
	return nil
}

func parsePresenceString(val string) user.Presence {
	parsed, err := strconv.Atoi(val)
	if err != nil {
		return user.NewPresenceOffline()
	}

	p, err := user.ParsePresence(parsed)
	if err != nil {
		return user.NewPresenceOffline()
	}

	return p
}

func (c *PresenceCache) AddNode(ctx context.Context, userID, nodeID fields.ID) error {
	pKey := userPresenceKey(userID)
	nKey := userNodesKey(userID)

	_, err := c.client.Pipelined(ctx, func(pipe redisdriver.Pipeliner) error {
		pipe.SAdd(ctx, nKey, nodeID.String())
		pipe.Expire(ctx, nKey, userNodesTTL)
		pipe.Expire(ctx, pKey, userPresenceTTL)
		return nil
	})
	if err != nil {
		return redis.NewError(err, redis.ScopePresence)
	}
	return nil
}

func (c *PresenceCache) RemoveNode(ctx context.Context, userID, nodeID fields.ID) error {
	if err := c.client.SRem(ctx, userNodesKey(userID), nodeID.String()).Err(); err != nil {
		return redis.NewError(err, redis.ScopePresence)
	}
	return nil
}

func (c *PresenceCache) RemoveBatchNodes(ctx context.Context, userIDs []fields.ID, nodeID fields.ID) error {
	if len(userIDs) == 0 {
		return nil
	}

	for i := 0; i < len(userIDs); i += MaxBatchSize {
		if err := ctx.Err(); err != nil {
			return err
		}

		end := min(i+MaxBatchSize, len(userIDs))
		chunk := userIDs[i:end]

		_, err := c.client.Pipelined(ctx, func(pipe redisdriver.Pipeliner) error {
			for _, userID := range chunk {
				pipe.SRem(ctx, userNodesKey(userID), nodeID.String())
			}
			return nil
		})
		if err != nil {
			return redis.NewError(err, redis.ScopePresence)
		}
	}

	return nil
}

func (c *PresenceCache) GetBatchNodes(
	ctx context.Context,
	userIDs []fields.ID,
) (map[fields.ID][]fields.ID, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}

	nodeToUsers := make(map[fields.ID][]fields.ID)

	for i := 0; i < len(userIDs); i += MaxBatchSize {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		end := min(i+MaxBatchSize, len(userIDs))
		chunk := userIDs[i:end]

		cmds := make(map[fields.ID]*redisdriver.StringSliceCmd, len(chunk))

		_, err := c.client.Pipelined(ctx, func(pipe redisdriver.Pipeliner) error {
			for _, uid := range chunk {
				cmds[uid] = pipe.SMembers(ctx, userNodesKey(uid))
			}
			return nil
		})
		if err != nil && err != redisdriver.Nil {
			return nil, redis.NewError(err, redis.ScopePresence)
		}

		for uid, cmd := range cmds {
			nodeStrs, parseErr := cmd.Result()
			if parseErr != nil || len(nodeStrs) == 0 {
				continue
			}

			for _, nStr := range nodeStrs {
				if parsedUUID, parseErr := uuid.Parse(nStr); parseErr == nil {
					nid := fields.ID(parsedUUID)
					nodeToUsers[nid] = append(nodeToUsers[nid], uid)
				}
			}
		}
	}

	return nodeToUsers, nil
}

var heartbeatScript = redisdriver.NewScript(`
    -- Ensure node ID is present in the set
    redis.call("SADD", KEYS[1], ARGV[1])
    redis.call("EXPIRE", KEYS[1], ARGV[2])

    -- If presence key expired, reset it to default active state (e.g. 1 / Online)
    if redis.call("EXISTS", KEYS[2]) == 0 then
        redis.call("SET", KEYS[2], ARGV[4], "EX", ARGV[3])
    else
        redis.call("EXPIRE", KEYS[2], ARGV[3])
    end

    return 1
`)

func (c *PresenceCache) Heartbeat(ctx context.Context, userID fields.ID, nodeID fields.ID) error {
	nKey := userNodesKey(userID)
	pKey := userPresenceKey(userID)

	err := heartbeatScript.Run(
		ctx,
		c.client,
		[]string{nKey, pKey},
		nodeID.String(),
		int(userNodesTTL.Seconds()),
		int(userPresenceTTL.Seconds()),
	).Err()

	if err != nil {
		return redis.NewError(err, redis.ScopePresence)
	}
	return nil
}
