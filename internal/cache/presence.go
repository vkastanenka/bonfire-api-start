package cache

import (
	"context"
	"time"

	"bonfire-api/internal/fields"
	"bonfire-api/internal/presence"
	"bonfire-api/internal/redis"

	"github.com/google/uuid"
	redisdriver "github.com/redis/go-redis/v9"
)

var (
	userPresenceTTL = 90 * time.Second
	userNodesTTL    = 90 * time.Second
	sessionNodeTTL  = userNodesTTL
)

func userPresenceKey(id fields.ID) string { return userNamespacedKey(id, "presence") }
func userSessionsKey(id fields.ID) string { return userNamespacedKey(id, "sessions") }
func userNodesKey(id fields.ID) string    { return userNamespacedKey(id, "nodes") }
func sessionNodeKey(id fields.ID) string  { return sessionNamespacedKey(id, "node") }

var (
	// Atomic Registration: Updates sessions, presence, and node tracking in a single atomic thread.
	registerNodeScript = redisdriver.NewScript(`
		local pKey = KEYS[1]
		local sSetKey = KEYS[2]
		local sNodeKey = KEYS[3]
		local uNodesKey = KEYS[4]

		local sessionID = ARGV[1]
		local nodeID = ARGV[2]
		local targetStatus = ARGV[3]
		local ttl = tonumber(ARGV[4])

		-- Check session count before adding
		local activeSessions = redis.call('SCARD', sSetKey)
		
		redis.call('SADD', sSetKey, sessionID)
		redis.call('EXPIRE', sSetKey, ttl)

		redis.call('SET', sNodeKey, nodeID, 'EX', ttl)

		redis.call('SADD', uNodesKey, nodeID)
		redis.call('EXPIRE', uNodesKey, ttl)

		redis.call('SETNX', pKey, targetStatus)
		redis.call('EXPIRE', pKey, ttl)

		local wasOffline = (activeSessions == 0) and 1 or 0
		local currentStatus = redis.call('GET', pKey)

		return { wasOffline, currentStatus }
	`)

	// Atomic Unregistration: Safely checks active session bounds without pipeline races.
	unregisterNodeScript = redisdriver.NewScript(`
		local pKey = KEYS[1]
		local sSetKey = KEYS[2]
		local sNodeKey = KEYS[3]

		local sessionID = ARGV[1]
		local ttl = tonumber(ARGV[2])

		redis.call('DEL', sNodeKey)
		redis.call('SREM', sSetKey, sessionID)

		local remainingSessions = redis.call('SCARD', sSetKey)
		if remainingSessions == 0 then
			redis.call('SET', pKey, '0', 'EX', ttl)
			redis.call('DEL', sSetKey)
			return 1 -- wentOffline = true
		end

		return 0 -- wentOffline = false
	`)
)

type PresenceCache struct {
	client redisdriver.Cmdable
}

func NewPresenceCache(client redisdriver.Cmdable) *PresenceCache {
	return &PresenceCache{
		client: client,
	}
}

func (c *PresenceCache) GetPresence(ctx context.Context, userID fields.ID) (presence.Presence, error) {
	data, found, err := getKey(ctx, c.client, userPresenceKey(userID), redis.ScopePresence)
	if err != nil || !found {
		return presence.NewOffline(), err
	}

	return parsePresence(string(data)), nil
}

func (c *PresenceCache) GetBatchPresence(
	ctx context.Context,
	userIDs []fields.ID,
) (map[fields.ID]presence.Presence, error) {
	result := make(map[fields.ID]presence.Presence, len(userIDs))
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
				result[id] = presence.NewOffline()
				continue
			}

			result[id] = parsePresence(string(data))
		}
	}

	return result, nil
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

func (c *PresenceCache) GetSessionNode(ctx context.Context, sessionID fields.ID) (fields.ID, bool, error) {
	data, found, err := getKey(ctx, c.client, sessionNodeKey(sessionID), redis.ScopePresence)
	if err != nil || !found {
		return fields.ID{}, false, err
	}

	parsedUUID, err := uuid.Parse(string(data))
	if err != nil {
		return fields.ID{}, false, nil
	}

	return fields.ID(parsedUUID), true, nil
}

func (c *PresenceCache) RegisterNode(
	ctx context.Context,
	userID, nodeID, sessionID fields.ID,
	p presence.Presence,
) (bool, presence.Presence, error) {
	targetPresence := p
	if !targetPresence.IsValid() {
		targetPresence = presence.NewOnline()
	}

	keys := []string{
		userPresenceKey(userID),
		userSessionsKey(userID),
		sessionNodeKey(sessionID),
		userNodesKey(userID),
	}

	res, err := registerNodeScript.Run(
		ctx,
		c.client,
		keys,
		sessionID.String(),
		nodeID.String(),
		targetPresence.Int(),
		int(userPresenceTTL.Seconds()),
	).Slice()

	if err != nil {
		return false, presence.NewOffline(), redis.NewError(err, redis.ScopePresence)
	}

	wasOffline := res[0].(int64) == 1
	effPresence := parsePresence(res[1].(string))

	return wasOffline, effPresence, nil
}

func (c *PresenceCache) UnregisterNode(ctx context.Context, userID, nodeID, sessionID fields.ID) (bool, error) {
	keys := []string{
		userPresenceKey(userID),
		userSessionsKey(userID),
		sessionNodeKey(sessionID),
	}

	wentOffline, err := unregisterNodeScript.Run(
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

func (c *PresenceCache) Heartbeat(ctx context.Context, userID, nodeID, sessionID fields.ID) error {
	pKey := userPresenceKey(userID)
	sSetKey := userSessionsKey(userID)
	uNodesKey := userNodesKey(userID)
	sNodeKey := sessionNodeKey(sessionID)

	_, err := c.client.Pipelined(ctx, func(pipe redisdriver.Pipeliner) error {
		pipe.Expire(ctx, sSetKey, userPresenceTTL)
		pipe.Expire(ctx, pKey, userPresenceTTL)
		pipe.Expire(ctx, uNodesKey, userNodesTTL)
		pipe.Expire(ctx, sNodeKey, userPresenceTTL)
		return nil
	})
	if err != nil {
		return redis.NewError(err, redis.ScopePresence)
	}

	return nil
}
