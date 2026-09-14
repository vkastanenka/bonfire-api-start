package cache

import (
	"context"
	"fmt"
	"time"

	"bonfire-api/internal/presence"
	"bonfire-api/internal/redis"

	"github.com/google/uuid"
	redisdriver "github.com/redis/go-redis/v9"
)

var (
	userPresenceTTL = 90 * time.Second
	userSessionsTTL = 90 * time.Second
)

type PresenceCache struct {
	client redisdriver.Cmdable
}

func NewPresenceCache(client redisdriver.Cmdable) *PresenceCache {
	return &PresenceCache{
		client: client,
	}
}

func (c *PresenceCache) GetPresence(ctx context.Context, userID uuid.UUID) (presence.Presence, error) {
	data, found, err := getKey(ctx, c.client, userPresenceKey(userID), redis.ScopePresence)
	if err != nil || !found {
		return presence.PresenceOffline, err
	}

	return parsePresence(string(data)), nil
}

func (c *PresenceCache) GetBatchPresence(
	ctx context.Context,
	userIDs []uuid.UUID,
) (map[uuid.UUID]presence.Presence, error) {
	result := make(map[uuid.UUID]presence.Presence, len(userIDs))
	if len(userIDs) == 0 {
		return result, nil
	}

	for i := 0; i < len(userIDs); i += maxBatchSize {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		end := min(i+maxBatchSize, len(userIDs))
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
				result[id] = presence.PresenceOffline
				continue
			}

			result[id] = parsePresence(string(data))
		}
	}

	return result, nil
}

func (c *PresenceCache) SetPresence(ctx context.Context, userID uuid.UUID, p presence.Presence) error {
	if err := c.client.Set(ctx, userPresenceKey(userID), p, userPresenceTTL).Err(); err != nil {
		return redis.NewError(err, redis.ScopePresence)
	}
	return nil
}

func (c *PresenceCache) GetSessionNode(
	ctx context.Context,
	userID, sessionID uuid.UUID,
) (uuid.UUID, bool, error) {
	nodeIDStr, err := c.client.HGet(ctx, userSessionsKey(userID), sessionID.String()).Result()
	if err == redisdriver.Nil {
		return uuid.UUID{}, false, nil
	}
	if err != nil {
		return uuid.UUID{}, false, redis.NewError(err, redis.ScopePresence)
	}

	parsedUUID, err := uuid.Parse(nodeIDStr)
	if err != nil {
		return uuid.UUID{}, false, nil
	}

	return uuid.UUID(parsedUUID), true, nil
}

var registerNodeSessionScript = redisdriver.NewScript(`
		local pKey = KEYS[1]
		local sHashKey = KEYS[2]

		local nodeID = ARGV[1]
		local sessionID = ARGV[2]
		local targetStatus = ARGV[3]
		local ttl = tonumber(ARGV[4])

		-- Count active sessions before adding
		local activeSessions = redis.call('HLEN', sHashKey)

		-- Track session in the user's hash
		redis.call('HSET', sHashKey, sessionID, nodeID)
		redis.call('EXPIRE', sHashKey, ttl)

		-- Only set presence if missing (SETNX) so explicit states aren't overwritten
		redis.call('SETNX', pKey, targetStatus)
		redis.call('EXPIRE', pKey, ttl)

		local wasOffline = (activeSessions == 0) and 1 or 0
		local currentStatus = redis.call('GET', pKey)

		return { wasOffline, currentStatus }
	`)

func (c *PresenceCache) RegisterNodeSession(
	ctx context.Context,
	nodeID, userID, sessionID uuid.UUID,
	p presence.Presence,
) (bool, presence.Presence, error) {
	targetPresence := p
	if !targetPresence.IsValid() {
		targetPresence = presence.PresenceOnline
	}

	keys := []string{
		userPresenceKey(userID),
		userSessionsKey(userID),
	}

	res, err := registerNodeSessionScript.Run(
		ctx,
		c.client,
		keys,
		nodeID.String(),
		sessionID.String(),
		targetPresence,
		int(userPresenceTTL.Seconds()),
	).Slice()

	if err != nil {
		return false, presence.PresenceOffline, redis.NewError(err, redis.ScopePresence)
	}

	wasOffline := res[0].(int64) == 1
	effPresence := parsePresence(fmt.Sprintf("%v", res[1]))

	return wasOffline, effPresence, nil
}

var unregisterNodeSessionScript = redisdriver.NewScript(`
		local pKey = KEYS[1]
		local sHashKey = KEYS[2]

		local sessionID = ARGV[1]
		local ttl = tonumber(ARGV[2])

		redis.call('HDEL', sHashKey, sessionID)

		local remainingSessions = redis.call('HLEN', sHashKey)
		if remainingSessions == 0 then
			redis.call('DEL', pKey)
			redis.call('DEL', sHashKey)
			return 1 -- wentOffline = true
		end

		redis.call('EXPIRE', sHashKey, ttl)
		redis.call('EXPIRE', pKey, ttl)

		return 0 -- wentOffline = false
	`)

func (c *PresenceCache) UnregisterNodeSession(ctx context.Context, nodeID, userID, sessionID uuid.UUID) (bool, error) {
	keys := []string{
		userPresenceKey(userID),
		userSessionsKey(userID),
	}

	wentOffline, err := unregisterNodeSessionScript.Run(
		ctx,
		c.client,
		keys,
		sessionID.String(),
		int(userPresenceTTL.Seconds()),
	).Int64()

	if err != nil {
		return false, redis.NewError(err, redis.ScopePresence)
	}

	return wentOffline == 1, nil
}

func (c *PresenceCache) Heartbeat(ctx context.Context, nodeID, userID, sessionID uuid.UUID) error {
	pKey := userPresenceKey(userID)
	sHashKey := userSessionsKey(userID)

	_, err := c.client.Pipelined(ctx, func(pipe redisdriver.Pipeliner) error {
		pipe.Expire(ctx, sHashKey, userSessionsTTL)
		pipe.Expire(ctx, pKey, userPresenceTTL)
		return nil
	})
	if err != nil {
		return redis.NewError(err, redis.ScopePresence)
	}

	return nil
}

func (c *PresenceCache) GetBatchNodeUsers(
	ctx context.Context,
	userIDs []uuid.UUID,
) (map[uuid.UUID][]uuid.UUID, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}

	nodeToUsers := make(map[uuid.UUID][]uuid.UUID)

	for i := 0; i < len(userIDs); i += maxBatchSize {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		end := min(i+maxBatchSize, len(userIDs))
		chunk := userIDs[i:end]
		cmds := make([]*redisdriver.MapStringStringCmd, len(chunk))

		_, err := c.client.Pipelined(ctx, func(pipe redisdriver.Pipeliner) error {
			for j, uid := range chunk {
				cmds[j] = pipe.HGetAll(ctx, userSessionsKey(uid))
			}
			return nil
		})
		if err != nil {
			return nil, redis.NewError(err, redis.ScopePresence)
		}

		for j, cmd := range cmds {
			uid := chunk[j]
			sessionMap, cmdErr := cmd.Result()
			if cmdErr != nil || len(sessionMap) == 0 {
				continue
			}

			nodeSet := make(map[uuid.UUID]struct{})
			for _, nodeIDStr := range sessionMap {
				if parsedUUID, parseErr := uuid.Parse(nodeIDStr); parseErr == nil {
					nodeSet[uuid.UUID(parsedUUID)] = struct{}{}
				}
			}

			for nID := range nodeSet {
				nodeToUsers[nID] = append(nodeToUsers[nID], uid)
			}
		}
	}

	return nodeToUsers, nil
}

var removeBatchNodeUsersScript = redisdriver.NewScript(`
		local pKey = KEYS[1]
		local sHashKey = KEYS[2]
		local targetNodeID = ARGV[1]
		local ttl = tonumber(ARGV[2])

		local sessions = redis.call('HGETALL', sHashKey)
		for i = 1, #sessions, 2 do
			local sessID = sessions[i]
			local nodeID = sessions[i+1]
			if nodeID == targetNodeID then
				redis.call('HDEL', sHashKey, sessID)
			end
		end

		local remainingSessions = redis.call('HLEN', sHashKey)
		if remainingSessions == 0 then
			redis.call('DEL', pKey)
			redis.call('DEL', sHashKey)
			return 1 -- wentOffline = true
		end

		redis.call('EXPIRE', sHashKey, ttl)
		redis.call('EXPIRE', pKey, ttl)

		return 0 -- wentOffline = false
	`)

func (c *PresenceCache) RemoveBatchNodeUsers(ctx context.Context, nodeID uuid.UUID, userIDs []uuid.UUID) error {
	if len(userIDs) == 0 {
		return nil
	}

	for i := 0; i < len(userIDs); i += maxBatchSize {
		if err := ctx.Err(); err != nil {
			return err
		}

		end := min(i+maxBatchSize, len(userIDs))
		chunk := userIDs[i:end]

		_, err := c.client.Pipelined(ctx, func(pipe redisdriver.Pipeliner) error {
			for _, userID := range chunk {
				keys := []string{
					userPresenceKey(userID),
					userSessionsKey(userID),
				}
				removeBatchNodeUsersScript.Run(ctx, pipe, keys, nodeID.String(), int(userPresenceTTL.Seconds()))
			}
			return nil
		})
		if err != nil {
			return redis.NewError(err, redis.ScopePresence)
		}
	}

	return nil
}
