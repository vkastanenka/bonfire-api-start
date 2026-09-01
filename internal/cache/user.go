package cache

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"bonfire-api/internal/errs"
	"bonfire-api/internal/fields"
	"bonfire-api/internal/redis"
	"bonfire-api/internal/user"

	"github.com/google/uuid"
	redisdriver "github.com/redis/go-redis/v9"
)

var (
	userTTL         = 24 * time.Hour
	userPresenceTTL = 90 * time.Second
	userNodesTTL    = 90 * time.Second
	userFriendsTTL  = 24 * time.Hour
	userChannelsTTL = 24 * time.Hour
)

const (
	userDomainKey = "user:"
)

func userNamespacedKey(id fields.ID, suffix string) string {
	if suffix == "" {
		return "{" + userDomainKey + id.String() + "}"
	}
	return "{" + userDomainKey + id.String() + "}:" + suffix
}

func userKey(id fields.ID) string               { return userNamespacedKey(id, "") }
func userPresenceKey(id fields.ID) string       { return userNamespacedKey(id, "presence") }
func userSessionsKey(id fields.ID) string       { return userNamespacedKey(id, "sessions") }
func userNodesKey(id fields.ID) string          { return userNamespacedKey(id, "nodes") }
func userFriendsKey(id fields.ID) string        { return userNamespacedKey(id, "friends") }
func userFriendRequestsKey(id fields.ID) string { return userNamespacedKey(id, "friend_requests") }
func userBlocksKey(id fields.ID) string         { return userNamespacedKey(id, "blocks") }
func userChannelsKey(id fields.ID) string       { return userNamespacedKey(id, "channels") }

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

// --- Redis Scripts ---

var registerWSConnectionScript = redisdriver.NewScript(`
    -- KEYS[1]: user:nodes set
    -- KEYS[2]: user:presence key
    -- ARGV[1]: connID (unique per websocket connection)
    -- ARGV[2]: nodeTTL
    -- ARGV[3]: initialPresence
    -- ARGV[4]: presenceTTL

    -- Check if user was completely offline (no active connections in set)
    local wasOffline = redis.call("SCARD", KEYS[1]) == 0

    -- Add unique connection ID to set
    redis.call("SADD", KEYS[1], ARGV[1])
    redis.call("EXPIRE", KEYS[1], ARGV[2])

    -- Set presence state ONLY if absent (NX)
    local setResult = redis.call("SET", KEYS[2], ARGV[3], "EX", ARGV[4], "NX")
    if not setResult then
        redis.call("EXPIRE", KEYS[2], ARGV[4])
    end

    local currentPresence = redis.call("GET", KEYS[2])
    return { wasOffline and 1 or 0, tonumber(currentPresence) }
`)

var unregisterScript = redisdriver.NewScript(`
    -- KEYS[1]: user:nodes set
    -- KEYS[2]: user:presence key
    -- ARGV[1]: connID
    -- ARGV[2]: offlinePresence
    -- ARGV[3]: presenceTTL
    -- ARGV[4]: nodeTTL

    redis.call("SREM", KEYS[1], ARGV[1])
    local count = redis.call("SCARD", KEYS[1])
    
    if count == 0 then
        -- User has zero active sockets remaining across all nodes
        redis.call("SET", KEYS[2], ARGV[2], "EX", ARGV[3])
        return 1 -- Went completely offline
    else
        redis.call("EXPIRE", KEYS[1], ARGV[4])
        redis.call("EXPIRE", KEYS[2], ARGV[3])
    end
    return 0 -- Active connections remaining
`)

var heartbeatScript = redisdriver.NewScript(`
	redis.call("SADD", KEYS[1], ARGV[1])
	redis.call("EXPIRE", KEYS[1], ARGV[2])
	redis.call("EXPIRE", KEYS[2], ARGV[3])
	return 1
`)

// --- user ---

func (u *UserCache) Get(ctx context.Context, id fields.ID) (*user.User, error) {
	redisKey := userKey(id)

	data, err := u.client.Get(ctx, redisKey).Bytes()
	if redis.IsCacheMiss(err) {
		return nil, nil
	}
	if err != nil {
		return nil, redis.NewError(err, u.scope)
	}

	var dto User
	if err := json.Unmarshal(data, &dto); err != nil {
		_ = u.client.Del(ctx, redisKey).Err()
		return nil, nil
	}

	usr, err := dto.ToDomain()
	if err != nil {
		_ = u.client.Del(ctx, redisKey).Err()
		return nil, nil
	}

	return usr, nil
}

func (u *UserCache) GetBatch(
	ctx context.Context,
	ids []fields.ID,
) (map[fields.ID]*user.User, []fields.ID, error) {
	if len(ids) == 0 {
		return make(map[fields.ID]*user.User), nil, nil
	}

	found := make(map[fields.ID]*user.User, len(ids))
	missing := make([]fields.ID, 0, len(ids))

	for i := 0; i < len(ids); i += MaxBatchSize {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}

		end := min(i+MaxBatchSize, len(ids))
		chunk := ids[i:end]

		redisKeys := make([]string, len(chunk))
		for j, id := range chunk {
			redisKeys[j] = userKey(id)
		}

		vals, err := u.client.MGet(ctx, redisKeys...).Result()
		if err != nil {
			return nil, nil, redis.NewError(err, u.scope)
		}

		for j, raw := range vals {
			id := chunk[j]

			if raw == nil {
				missing = append(missing, id)
				continue
			}

			var data []byte
			switch v := raw.(type) {
			case string:
				if v == "" {
					missing = append(missing, id)
					continue
				}
				data = []byte(v)
			case []byte:
				if len(v) == 0 {
					missing = append(missing, id)
					continue
				}
				data = v
			default:
				missing = append(missing, id)
				continue
			}

			var dto User
			if err := json.Unmarshal(data, &dto); err != nil {
				missing = append(missing, id)
				continue
			}

			usr, err := dto.ToDomain()
			if err != nil {
				missing = append(missing, id)
				continue
			}

			found[id] = usr
		}
	}

	return found, missing, nil
}

func (u *UserCache) Set(ctx context.Context, usr *user.User) error {
	redisKey := userKey(usr.ID())
	dto := ParseUser(usr)

	bytes, err := json.Marshal(dto)
	if err != nil {
		return errs.Internal("Failed to marshal user json.").
			Meta("scope", u.scope.String()).
			Wrap(err)
	}

	if err := u.client.Set(ctx, redisKey, bytes, userTTL).Err(); err != nil {
		return redis.NewError(err, u.scope)
	}

	return nil
}

func (u *UserCache) SetBatch(ctx context.Context, users map[fields.ID]*user.User) error {
	if len(users) == 0 {
		return nil
	}

	type entry struct {
		id  fields.ID
		usr *user.User
	}
	validEntries := make([]entry, 0, len(users))
	for id, usr := range users {
		if usr != nil && !id.IsZero() {
			validEntries = append(validEntries, entry{id: id, usr: usr})
		}
	}

	if len(validEntries) == 0 {
		return nil
	}

	for i := 0; i < len(validEntries); i += MaxBatchSize {
		if err := ctx.Err(); err != nil {
			return err
		}

		end := min(i+MaxBatchSize, len(validEntries))
		chunk := validEntries[i:end]

		pipe := u.client.Pipeline()
		for _, e := range chunk {
			data, err := json.Marshal(ParseUser(e.usr))
			if err != nil {
				return errs.Internal("Failed to marshal user json.").
					Meta("scope", u.scope.String()).
					Wrap(err)
			}
			pipe.Set(ctx, userKey(e.id), data, userTTL)
		}

		if _, err := pipe.Exec(ctx); err != nil {
			return redis.NewError(err, u.scope)
		}
	}

	return nil
}

func (u *UserCache) Delete(ctx context.Context, id fields.ID) error {
	if err := u.client.Del(ctx, userKey(id)).Err(); err != nil {
		return redis.NewError(err, u.scope)
	}

	return nil
}

func (u *UserCache) DeleteBatch(ctx context.Context, ids []fields.ID) error {
	if len(ids) == 0 {
		return nil
	}

	for i := 0; i < len(ids); i += MaxBatchSize {
		if err := ctx.Err(); err != nil {
			return err
		}

		end := min(i+MaxBatchSize, len(ids))
		chunk := ids[i:end]

		redisKeys := make([]string, len(chunk))
		for j, id := range chunk {
			redisKeys[j] = userKey(id)
		}

		if err := u.client.Del(ctx, redisKeys...).Err(); err != nil {
			return redis.NewError(err, u.scope)
		}
	}

	return nil
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
	if err := c.client.SRem(ctx, userNodesKey(userID), nodeID.String()).Err(); err != nil {
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

// GetBatchNodes maps each recipient user ID to the gateway node(s) holding their active connection.
// Returns map[nodeID][]userID so broadcasts can be strictly targeted per node.
func (c *UserCache) GetBatchNodes(
	ctx context.Context,
	userIDs []fields.ID,
) (map[fields.ID][]fields.ID, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}

	cmds := make(map[fields.ID]*redisdriver.StringSliceCmd, len(userIDs))

	// Pipeline SMembers for {user:ID}:nodes across all recipients
	_, err := c.client.Pipelined(ctx, func(pipe redisdriver.Pipeliner) error {
		for _, uid := range userIDs {
			cmds[uid] = pipe.SMembers(ctx, userNodesKey(uid))
		}
		return nil
	})
	if err != nil && !errors.Is(err, redisdriver.Nil) {
		return nil, redis.NewError(err, c.scope)
	}

	nodeToUsers := make(map[fields.ID][]fields.ID)

	for uid, cmd := range cmds {
		nodeStrs, parseErr := cmd.Result()
		if parseErr != nil || len(nodeStrs) == 0 {
			continue
		}

		for _, nStr := range nodeStrs {
			if nodeID, parseErr := uuid.Parse(nStr); parseErr == nil {
				nid := fields.ID(nodeID)
				nodeToUsers[nid] = append(nodeToUsers[nid], uid)
			}
		}
	}

	return nodeToUsers, nil
}

func (c *UserCache) Heartbeat(ctx context.Context, userID fields.ID, nodeID fields.ID) error {
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
		return redis.NewError(err, c.scope)
	}
	return nil
}

type RegisterWSResult struct {
	WasOffline bool
	Presence   user.Presence
}

func (c *UserCache) RegisterWSConnection(ctx context.Context, userID, connID fields.ID, presence user.Presence) (RegisterWSResult, error) {
	nKey := userNodesKey(userID)
	pKey := userPresenceKey(userID)

	targetPresence := presence
	if !targetPresence.IsValid() {
		targetPresence = user.NewPresenceOnline()
	}

	res, err := registerWSConnectionScript.Run(
		ctx,
		c.client,
		[]string{nKey, pKey},
		connID.String(),                // ARGV[1] -> Unique connection identifier
		int(userNodesTTL.Seconds()),    // ARGV[2]
		targetPresence.Int(),           // ARGV[3]
		int(userPresenceTTL.Seconds()), // ARGV[4]
	).Slice()

	if err != nil {
		return RegisterWSResult{}, redis.NewError(err, c.scope)
	}

	wasOffline := res[0].(int64) == 1
	effPresence, err := user.ParsePresence(int(res[1].(int64)))
	if err != nil {
		return RegisterWSResult{}, err
	}

	return RegisterWSResult{
		WasOffline: wasOffline,
		Presence:   effPresence,
	}, nil
}

func (c *UserCache) UnregisterWSConnection(ctx context.Context, userID, connID fields.ID) (bool, error) {
	nKey := userNodesKey(userID)
	pKey := userPresenceKey(userID)

	res, err := unregisterScript.Run(
		ctx,
		c.client,
		[]string{nKey, pKey},
		connID.String(),                 // ARGV[1] -> Matches register SADD
		user.NewPresenceOffline().Int(), // ARGV[2]
		int(userPresenceTTL.Seconds()),  // ARGV[3]
		int(userNodesTTL.Seconds()),     // ARGV[4]
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
func (c *UserCache) SetFriends(ctx context.Context, userID fields.ID, friendIDs []fields.ID) error {
	fKey := userFriendsKey(userID)

	_, err := c.client.Pipelined(ctx, func(pipe redisdriver.Pipeliner) error {
		pipe.Del(ctx, fKey) // Clear existing cache first if overwriting
		if len(friendIDs) > 0 {
			members := make([]interface{}, len(friendIDs))
			for i, id := range friendIDs {
				members[i] = id.String()
			}
			pipe.SAdd(ctx, fKey, members...)
			pipe.Expire(ctx, fKey, userFriendsTTL)
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

func (c *UserCache) SetChannels(ctx context.Context, userID fields.ID, channelIDs []fields.ID) error {
	chKey := userChannelsKey(userID)

	_, err := c.client.Pipelined(ctx, func(pipe redisdriver.Pipeliner) error {
		pipe.Del(ctx, chKey)
		if len(channelIDs) > 0 {
			members := make([]interface{}, len(channelIDs))
			for i, id := range channelIDs {
				members[i] = id.String()
			}
			pipe.SAdd(ctx, chKey, members...)
			pipe.Expire(ctx, chKey, userChannelsTTL)
		}
		return nil
	})
	if err != nil {
		return redis.NewError(err, c.scope)
	}
	return nil
}

const maxChannelsToQuery = 100

// GetUpdateRecipients fetches deduplicated recipient user IDs across
// a user's cached friends list and active channel members in at most 2 RTTs.
func (c *UserCache) GetUpdateRecipients(
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
	if err != nil && !errors.Is(err, redisdriver.Nil) {
		return nil, redis.NewError(err, c.scope)
	}

	friendStrs, _ := friendsCmd.Result()
	channelStrs, _ := channelsCmd.Result()

	if len(friendStrs) == 0 && len(channelStrs) == 0 {
		return nil, nil
	}

	// Protect against payload/memory explosions for extreme power-users
	if len(channelStrs) > maxChannelsToQuery {
		channelStrs = channelStrs[:maxChannelsToQuery]
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
		if err != nil && !errors.Is(err, redisdriver.Nil) {
			return nil, redis.NewError(err, c.scope)
		}
	}

	// Initialize map with capacity for friends; map will grow if channels contribute new IDs
	recipientMap := make(map[fields.ID]struct{}, len(friendStrs))

	// Parse friends directly into fields.ID
	for _, fStr := range friendStrs {
		if id, parseErr := uuid.Parse(fStr); parseErr == nil && id != uuid.UUID(userID) {
			recipientMap[fields.ID(id)] = struct{}{}
		}
	}

	// Single-pass extraction over channel member commands
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

	// Extract unique recipient keys
	recipients := make([]fields.ID, 0, len(recipientMap))
	for id := range recipientMap {
		recipients = append(recipients, id)
	}

	return recipients, nil
}

// GetFriendNodes resolves all active gateway node mappings for a user's friends in 1 RTT pipeline.
func (c *UserCache) GetFriendNodes(
	ctx context.Context,
	userID fields.ID,
) (map[fields.ID][]fields.ID, error) {
	friends, err := c.GetFriends(ctx, userID)
	if err != nil || len(friends) == 0 {
		return nil, err
	}

	return c.GetBatchNodes(ctx, friends)
}

// GetUpdateRecipientNodes resolves active gateway node mappings for all update recipients
// (friends + active channel members) in 2 RTTs total.
func (c *UserCache) GetUpdateRecipientNodes(
	ctx context.Context,
	userID fields.ID,
) (map[fields.ID][]fields.ID, error) {
	recipients, err := c.GetUpdateRecipients(ctx, userID)
	if err != nil || len(recipients) == 0 {
		return nil, err
	}

	return c.GetBatchNodes(ctx, recipients)
}
