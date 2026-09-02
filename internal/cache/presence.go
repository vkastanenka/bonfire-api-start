package cache

import (
	"context"

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
	data, found, err := getKey(ctx, c.client, userPresenceKey(userID), redis.ScopePresence)
	if err != nil || !found {
		return user.NewPresenceOffline(), err
	}

	return parsePresence(string(data)), nil
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

		vals, err := getBatchKeys(ctx, c.client, redisKeys, redis.ScopePresence)
		if err != nil {
			return nil, err
		}

		for j, raw := range vals {
			id := chunk[j]

			data, ok := toBytes(raw)
			if !ok {
				result[id] = user.NewPresenceOffline()
				continue
			}

			result[id] = parsePresence(string(data))
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

func (c *PresenceCache) Heartbeat(ctx context.Context, userID, nodeID fields.ID) error {
	nKey := userNodesKey(userID)
	pKey := userPresenceKey(userID)

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
		cmds := make([]*redisdriver.StringSliceCmd, len(chunk))

		_, err := c.client.Pipelined(ctx, func(pipe redisdriver.Pipeliner) error {
			for j, uid := range chunk {
				cmds[j] = pipe.SMembers(ctx, userNodesKey(uid))
			}
			return nil
		})
		if err != nil {
			return nil, redis.NewError(err, redis.ScopePresence)
		}

		for j, cmd := range cmds {
			uid := chunk[j]
			nodeStrs, cmdErr := cmd.Result()
			if cmdErr != nil || len(nodeStrs) == 0 {
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

var registerNodeScript = redisdriver.NewScript(`
	-- KEYS[1]: user:nodes set
	-- KEYS[2]: user:presence key
	-- ARGV[1]: nodeID (unique gateway node identifier)
	-- ARGV[2]: nodeTTL
	-- ARGV[3]: initialPresence
	-- ARGV[4]: presenceTTL

	local wasOffline = redis.call("SCARD", KEYS[1]) == 0

	redis.call("SADD", KEYS[1], ARGV[1])
	redis.call("EXPIRE", KEYS[1], ARGV[2])

	local setResult = redis.call("SET", KEYS[2], ARGV[3], "EX", ARGV[4], "NX")
	if not setResult then
		redis.call("EXPIRE", KEYS[2], ARGV[4])
	end

	local currentPresence = redis.call("GET", KEYS[2])
	return { wasOffline and 1 or 0, tonumber(currentPresence) }
`)

func (c *PresenceCache) RegisterNode(
	ctx context.Context,
	userID, nodeID fields.ID,
	presence user.Presence,
) (bool, user.Presence, error) {
	nKey := userNodesKey(userID)
	pKey := userPresenceKey(userID)

	targetPresence := presence
	if !targetPresence.IsValid() {
		targetPresence = user.NewPresenceOnline()
	}

	res, err := registerNodeScript.Run(
		ctx,
		c.client,
		[]string{nKey, pKey},
		nodeID.String(),
		int(userNodesTTL.Seconds()),
		targetPresence.Int(),
		int(userPresenceTTL.Seconds()),
	).Slice()

	if err != nil {
		return false, user.NewPresenceOffline(), redis.NewError(err, redis.ScopePresence)
	}

	wasOffline := res[0].(int64) == 1
	effPresence, err := user.ParsePresence(int(res[1].(int64)))
	if err != nil {
		return false, user.NewPresenceOffline(), err
	}

	return wasOffline, effPresence, nil
}

var unregisterNodeScript = redisdriver.NewScript(`
	-- KEYS[1]: user:nodes set
	-- KEYS[2]: user:presence key
	-- ARGV[1]: nodeID
	-- ARGV[2]: offlinePresence
	-- ARGV[3]: presenceTTL
	-- ARGV[4]: nodeTTL

	redis.call("SREM", KEYS[1], ARGV[1])
	local count = redis.call("SCARD", KEYS[1])

	if count == 0 then
		local currentPresence = redis.call("GET", KEYS[2])
		-- If presence is missing or not already offline, set to offline and signal state change
		if not currentPresence or tonumber(currentPresence) ~= tonumber(ARGV[2]) then
			redis.call("SET", KEYS[2], ARGV[2], "EX", ARGV[3])
			return 1
		end
		-- Already offline
		return 0
	else
		redis.call("EXPIRE", KEYS[1], ARGV[4])
		redis.call("EXPIRE", KEYS[2], ARGV[3])
	end
	return 0
`)

func (c *PresenceCache) UnregisterNode(ctx context.Context, userID, nodeID fields.ID) (bool, error) {
	nKey := userNodesKey(userID)
	pKey := userPresenceKey(userID)

	res, err := unregisterNodeScript.Run(
		ctx,
		c.client,
		[]string{nKey, pKey},
		nodeID.String(),
		user.NewPresenceOffline().Int(),
		int(userPresenceTTL.Seconds()),
		int(userNodesTTL.Seconds()),
	).Int()

	if err != nil {
		return false, redis.NewError(err, redis.ScopePresence)
	}

	return res == 1, nil
}
