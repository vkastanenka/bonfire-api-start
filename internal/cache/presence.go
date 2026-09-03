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
	pKey := userPresenceKey(userID)
	sSetKey := userSessionsKey(userID)
	sNodeKey := sessionNodeKey(sessionID)

	targetPresence := p
	if !targetPresence.IsValid() {
		targetPresence = presence.NewOnline()
	}

	var scardCmd *redisdriver.IntCmd
	var getPresenceCmd *redisdriver.StringCmd

	_, err := c.client.Pipelined(ctx, func(pipe redisdriver.Pipeliner) error {
		scardCmd = pipe.SCard(ctx, sSetKey)
		pipe.SAdd(ctx, sSetKey, sessionID.String())
		pipe.Expire(ctx, sSetKey, userPresenceTTL)

		pipe.Set(ctx, sNodeKey, nodeID.String(), sessionNodeTTL)

		// Set status if missing, or refresh TTL if present
		pipe.SetNX(ctx, pKey, targetPresence.Int(), userPresenceTTL)
		pipe.Expire(ctx, pKey, userPresenceTTL)
		getPresenceCmd = pipe.Get(ctx, pKey)
		return nil
	})
	if err != nil && err != redisdriver.Nil {
		return false, presence.NewOffline(), redis.NewError(err, redis.ScopePresence)
	}

	wasOffline := scardCmd.Val() == 0
	effPresence, parseErr := presence.ParseString(getPresenceCmd.Val())
	if parseErr != nil {
		effPresence = targetPresence
	}

	return wasOffline, effPresence, nil
}

func (c *PresenceCache) UnregisterNode(ctx context.Context, userID, nodeID, sessionID fields.ID) (bool, error) {
	pKey := userPresenceKey(userID)
	sSetKey := userSessionsKey(userID)
	sNodeKey := sessionNodeKey(sessionID)

	var scardCmd *redisdriver.IntCmd

	_, err := c.client.Pipelined(ctx, func(pipe redisdriver.Pipeliner) error {
		pipe.Del(ctx, sNodeKey)
		pipe.SRem(ctx, sSetKey, sessionID.String())
		scardCmd = pipe.SCard(ctx, sSetKey)
		return nil
	})
	if err != nil {
		return false, redis.NewError(err, redis.ScopePresence)
	}

	// If no sessions remain for this user, mark them offline
	if scardCmd.Val() == 0 {
		_, err := c.client.Pipelined(ctx, func(pipe redisdriver.Pipeliner) error {
			pipe.Set(ctx, pKey, presence.NewOffline().Int(), userPresenceTTL)
			pipe.Del(ctx, sSetKey)
			return nil
		})
		if err != nil {
			return false, redis.NewError(err, redis.ScopePresence)
		}
		return true, nil // User went completely offline
	}

	return false, nil // User still has active sessions
}

func (c *PresenceCache) Heartbeat(ctx context.Context, userID, nodeID, sessionID fields.ID) error {
	pKey := userPresenceKey(userID)
	sSetKey := userSessionsKey(userID)
	sNodeKey := sessionNodeKey(sessionID)

	_, err := c.client.Pipelined(ctx, func(pipe redisdriver.Pipeliner) error {
		pipe.Expire(ctx, sSetKey, userPresenceTTL)
		pipe.Expire(ctx, pKey, userPresenceTTL)
		pipe.Expire(ctx, sNodeKey, userPresenceTTL)
		return nil
	})
	if err != nil {
		return redis.NewError(err, redis.ScopePresence)
	}

	return nil
}
