package cache

import (
	"bonfire-api/internal/pkg/helpers"
	"bonfire-api/internal/presence"
	"bonfire-api/internal/redis"
	"context"
	"time"

	"github.com/google/uuid"
	redisdriver "github.com/redis/go-redis/v9"
)

var (
	userSessionsTTL = 90 * time.Second
)

func GatewayEventsKey(id uuid.UUID) string { return "gateway:" + id.String() + ":events" }
func userSessionsKey(id uuid.UUID) string  { return "{user:" + id.String() + "}:sessions" }

type GatewayCache struct {
	client redisdriver.Cmdable
}

func NewGatewayCache(client redisdriver.Cmdable) *GatewayCache {
	return &GatewayCache{client: client}
}

func (c *GatewayCache) GetUserSessions(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, []uuid.UUID, map[uuid.UUID]uuid.UUID, error) {
	return getHashMapKeyVals(ctx, redis.ScopeGateway, c.client, userSessionsKey(userID), helpers.StringToUUID)
}

func (c *GatewayCache) SetUserSessions(ctx context.Context, userID uuid.UUID, sessions map[uuid.UUID]uuid.UUID) error {
	return setHashMapKeyVals(ctx, redis.ScopeGateway, c.client, userSessionsKey(userID), sessions, userSessionsTTL, helpers.UUIDToString)
}

func (c *GatewayCache) Heartbeat(ctx context.Context, nodeID, userID, sessionID uuid.UUID) error {
	pKey := userPresenceKey(userID)
	sHashKey := userSessionsKey(userID)

	_, err := c.client.Pipelined(ctx, func(pipe redisdriver.Pipeliner) error {
		pipe.Expire(ctx, sHashKey, userSessionsTTL)
		pipe.Expire(ctx, pKey, userPresenceTTL)
		return nil
	})
	if err != nil {
		return redis.NewError(err, redis.ScopeGateway)
	}

	return nil
}

var registerSessionScript = redisdriver.NewScript(`
		local pKey = KEYS[1]
		local sHashKey = KEYS[2]

		local nodeID = ARGV[1]
		local sessionID = ARGV[2]
		local targetStatus = ARGV[3]
		local ttl = tonumber(ARGV[4])

		-- Check session count before adding
		local activeSessions = redis.call('HLEN', sHashKey)

		-- Track session and refresh hash TTL
		redis.call('HSET', sHashKey, sessionID, nodeID)
		redis.call('EXPIRE', sHashKey, ttl)

		-- Only set target status if presence key does not exist
		redis.call('SETNX', pKey, targetStatus)
		redis.call('EXPIRE', pKey, ttl)

		return (activeSessions == 0) and 1 or 0
	`)

func (c *GatewayCache) RegisterSession(
	ctx context.Context,
	nodeID, userID, sessionID uuid.UUID,
	p presence.Presence,
) (bool, error) {
	targetPresence := p
	if !targetPresence.IsValid() {
		targetPresence = presence.PresenceOnline
	}

	keys := []string{
		userPresenceKey(userID),
		userSessionsKey(userID),
	}

	res, err := registerSessionScript.Run(
		ctx,
		c.client,
		keys,
		nodeID.String(),
		sessionID.String(),
		targetPresence.Int(),
		int(userPresenceTTL.Seconds()),
	).Int64()
	if err != nil {
		return false, redis.NewError(err, redis.ScopeGateway)
	}

	wasOffline := res == 1
	return wasOffline, nil
}

var unregisterSessionScript = redisdriver.NewScript(`
		local pKey = KEYS[1]
		local sHashKey = KEYS[2]

		local sessionID = ARGV[1]
		local nodeID = ARGV[2]
		local ttl = tonumber(ARGV[3])

		-- Check if the session is currently assigned to this node
		local currentOwnerNode = redis.call('HGET', sHashKey, sessionID)

		-- If the session doesn't exist or has been re-registered to a different node, don't remove it
		if currentOwnerNode and currentOwnerNode == nodeID then
			redis.call('HDEL', sHashKey, sessionID)
		end

		-- Check remaining active sessions across all nodes
		local remainingSessions = redis.call('HLEN', sHashKey)

		if remainingSessions == 0 then
			redis.call('DEL', pKey)
			redis.call('DEL', sHashKey) -- Ensures clean removal if any orphaned state remains
			return 1 -- wentOffline = true
		end

		-- Refresh TTL for remaining active sessions
		redis.call('EXPIRE', sHashKey, ttl)
		redis.call('EXPIRE', pKey, ttl)

		return 0 -- wentOffline = false
	`)

func (c *GatewayCache) UnregisterSession(ctx context.Context, nodeID, userID, sessionID uuid.UUID) (bool, error) {
	keys := []string{
		userPresenceKey(userID),
		userSessionsKey(userID),
	}

	wentOffline, err := unregisterSessionScript.Run(
		ctx,
		c.client,
		keys,
		sessionID.String(),
		int(userPresenceTTL.Seconds()),
	).Int64()

	if err != nil {
		return false, redis.NewError(err, redis.ScopeGateway)
	}

	return wentOffline == 1, nil
}

func (c *GatewayCache) GetBatchUsers(
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

		keys := make([]string, len(chunk))
		for j, uid := range chunk {
			keys[j] = userSessionsKey(uid)
		}

		batchResults, err := getBatchHashMaps(ctx, redis.ScopeGateway, c.client, keys)
		if err != nil {
			return nil, err
		}

		for j, sessionMap := range batchResults {
			if len(sessionMap) == 0 {
				continue
			}

			userID := chunk[j]

			seenNodesForUser := make(map[uuid.UUID]struct{}, len(sessionMap))

			for _, nodeIDStr := range sessionMap {
				if nodeIDStr == "" {
					continue
				}

				nodeID, parseErr := uuid.Parse(nodeIDStr)
				if parseErr != nil {
					continue
				}

				if _, seen := seenNodesForUser[nodeID]; !seen {
					seenNodesForUser[nodeID] = struct{}{}
					nodeToUsers[nodeID] = append(nodeToUsers[nodeID], userID)
				}
			}
		}
	}

	return nodeToUsers, nil
}

var removeBatchUsersScript = redisdriver.NewScript(`
		local pKey = KEYS[1]
		local sHashKey = KEYS[2]
		local targetNodeID = ARGV[1]
		local ttl = tonumber(ARGV[2])

		-- Fast path: if session key doesn't exist, ensure presence key is cleaned up
		if redis.call('EXISTS', sHashKey) == 0 then
			redis.call('DEL', pKey)
			return
		end

		local sessions = redis.call('HGETALL', sHashKey)
		local toDelete = {}

		for i = 1, #sessions, 2 do
			local sessID = sessions[i]
			local nodeID = sessions[i+1]
			if nodeID == targetNodeID then
				table.insert(toDelete, sessID)
			end
		end

		-- Single variadic HDEL if matching sessions were found
		if #toDelete > 0 then
			redis.call('HDEL', sHashKey, unpack(toDelete))
		end

		-- If no sessions remain, purge presence & session keys
		local remainingSessions = redis.call('HLEN', sHashKey)
		if remainingSessions == 0 then
			redis.call('DEL', pKey)
			redis.call('DEL', sHashKey)
			return
		end

		-- Refresh TTL for remaining active sessions on other nodes
		redis.call('EXPIRE', sHashKey, ttl)
		redis.call('EXPIRE', pKey, ttl)
`)

func (c *GatewayCache) RemoveBatchUsers(
	ctx context.Context,
	nodeID uuid.UUID,
	userIDs []uuid.UUID,
) error {
	if len(userIDs) == 0 {
		return nil
	}

	nodeIDStr := nodeID.String()
	ttlSeconds := int(userPresenceTTL.Seconds())

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

				removeBatchUsersScript.Run(ctx, pipe, keys, nodeIDStr, ttlSeconds)
			}
			return nil
		})
		if err != nil {
			return redis.NewError(err, redis.ScopeGateway)
		}
	}

	return nil
}
