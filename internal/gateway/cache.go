package gateway

import (
	"context"
	"strconv"
	"time"

	"bonfire-api/internal/pkg/errs"
	"bonfire-api/internal/redis"

	"github.com/google/uuid"
	redisdriver "github.com/redis/go-redis/v9"
)

const (
	TicketCacheTTL = 30 * time.Second
	UserCacheTTL   = 90 * time.Second
)

func TicketCacheKey(ticketID uuid.UUID) string {
	return "gateway:ticket:" + ticketID.String()
}

func UserCacheKey(id uuid.UUID) string {
	return "gateway:user:" + "{" + id.String() + "}"
}

type Cache struct {
	client redisdriver.Cmdable
}

func NewCache(client redisdriver.Cmdable) *Cache {
	return &Cache{client: client}
}

func (c *Cache) PrintTicket(ctx context.Context, ticketID, userID, sessionID, clientID uuid.UUID) error {
	var payload [32]byte
	copy(payload[0:16], userID[:])
	copy(payload[16:32], sessionID[:])

	if err := c.client.Set(ctx, TicketCacheKey(ticketID), payload[:], TicketCacheTTL).Err(); err != nil {
		return redis.NewError(err, redis.ResourceGateway)
	}
	return nil
}

func (c *Cache) PunchTicket(ctx context.Context, ticketID uuid.UUID) (uuid.UUID, uuid.UUID, error) {
	val, err := c.client.GetDel(ctx, TicketCacheKey(ticketID)).Bytes()
	if err != nil {
		return uuid.UUID{}, uuid.UUID{}, redis.NewError(err, redis.ResourceGateway)
	}

	if len(val) != 32 {
		return uuid.UUID{}, uuid.UUID{}, errs.Internal("Invalid websocket ticket payload length.")
	}

	userID := uuid.Must(uuid.FromBytes(val[0:16]))
	sessionID := uuid.Must(uuid.FromBytes(val[16:32]))

	return userID, sessionID, nil
}

var registerClientScript = redisdriver.NewScript(`
	local userKey = KEYS[1]

	local hubID = ARGV[1]
	local sessionID = ARGV[2]
	local clientID = ARGV[3]
	local presence = ARGV[4]
	local ttl = tonumber(ARGV[5])

	-- Check if presence field already exists to determine if user was offline
	local wasOffline = (redis.call('HEXISTS', userKey, 'presence') == 0) and 1 or 0

	-- Construct composite field name: sessionID:clientID
	local fieldName = sessionID .. ':' .. clientID

	-- Set/update presence and add the composite session mapping in a single atomic hash
	redis.call('HSET', userKey, 'presence', presence, fieldName, hubID)
	redis.call('EXPIRE', userKey, ttl)

	return wasOffline
`)

func (c *Cache) RegisterClient(
	ctx context.Context,
	hubID, userID, sessionID, clientID uuid.UUID,
	p Presence,
) (bool, error) {
	parsedPresence := p
	if !parsedPresence.IsValid() {
		parsedPresence = PresenceOnline
	}

	keys := []string{
		UserCacheKey(userID),
	}

	res, err := registerClientScript.Run(
		ctx,
		c.client,
		keys,
		hubID.String(),
		sessionID.String(),
		clientID.String(),
		parsedPresence.Int(),
		int(UserCacheTTL.Seconds()),
	).Int64()
	if err != nil {
		return false, redis.NewError(err, redis.ResourceGateway)
	}

	wasOffline := res == 1
	return wasOffline, nil
}

var unregisterClientScript = redisdriver.NewScript(`
	local userKey = KEYS[1]

	local hubID = ARGV[1]
	local sessionID = ARGV[2]
	local clientID = ARGV[3]
	local ttl = tonumber(ARGV[4])

	local sessionField = sessionID .. ':' .. clientID

	-- Verify the session/client connection exists and belongs to this node before removing
	local currentOwnerNode = redis.call('HGET', userKey, sessionField)
	if not currentOwnerNode or currentOwnerNode ~= hubID then
		return 0 -- Connection not found or owned by a different node
	end

	-- Remove the specific client session mapping
	redis.call('HDEL', userKey, sessionField)

	-- If HLEN <= 1, only the 'presence' field remains (0 active connections)
	if redis.call('HLEN', userKey) <= 1 then
		redis.call('DEL', userKey)
		return 1 -- wentOffline = true
	end

	-- Otherwise, refresh the TTL for the remaining active connections and presence
	redis.call('EXPIRE', userKey, ttl)

	return 0 -- wentOffline = false
`)

func (c *Cache) UnregisterClient(
	ctx context.Context,
	hubID, userID, sessionID, clientID uuid.UUID,
) (bool, error) {
	keys := []string{
		UserCacheKey(userID),
	}

	res, err := unregisterClientScript.Run(
		ctx,
		c.client,
		keys,
		hubID.String(),
		sessionID.String(),
		clientID.String(),
		int(UserCacheTTL.Seconds()),
	).Int64()
	if err != nil {
		return false, redis.NewError(err, redis.ResourceGateway)
	}

	wentOffline := res == 1
	return wentOffline, nil
}

func (c *Cache) GetBatchUsers(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]map[string]string, error) {
	if len(userIDs) == 0 {
		return make(map[uuid.UUID]map[string]string), nil
	}

	pipe := c.client.Pipeline()
	cmds := make(map[uuid.UUID]*redisdriver.MapStringStringCmd, len(userIDs))

	for _, userID := range userIDs {
		cmds[userID] = pipe.HGetAll(ctx, UserCacheKey(userID))
	}

	_, err := pipe.Exec(ctx)
	if err != nil && err != redisdriver.Nil {
		return nil, redis.NewError(err, redis.ResourceGateway)
	}

	result := make(map[uuid.UUID]map[string]string, len(userIDs))
	for userID, cmd := range cmds {
		res, err := cmd.Result()
		if err != nil && err != redisdriver.Nil {
			continue
		}
		if len(res) > 0 {
			result[userID] = res
		}
	}

	return result, nil
}

var removeBatchClientsScript = redisdriver.NewScript(`
	local userKey = KEYS[1]
	local targetHubID = ARGV[1]
	local ttl = tonumber(ARGV[2])

	-- Get all fields and values from the user hash
	local hashEntries = redis.call('HGETALL', userKey)
	if #hashEntries == 0 then
		return 0
	end

	local fieldsToDelete = {}
	local hasOtherHubs = false

	-- HGETALL returns a flat list: {field1, value1, field2, value2, ...}
	for i = 1, #hashEntries, 2 do
		local field = hashEntries[i]
		local val = hashEntries[i+1]

		if field ~= 'presence' then
			if val == targetHubID then
				table.insert(fieldsToDelete, field)
			else
				hasOtherHubs = true
			end
		end
	end

	-- If we found fields belonging to this hub, remove them
	if #fieldsToDelete > 0 then
		redis.call('HDEL', userKey, unpack(fieldsToDelete))
	end

	-- Check remaining fields (excluding 'presence')
	local remainingLength = redis.call('HLEN', userKey)

	-- If only 'presence' remains (or 0 fields left), drop the user key entirely
	if remainingLength <= 1 then
		redis.call('DEL', userKey)
		return 1 -- User went offline completely
	else
		-- If the user still has sessions on other hubs, refresh TTL
		redis.call('EXPIRE', userKey, ttl)
		return 0
	end
`)

func (c *Cache) RemoveBatchClients(ctx context.Context, hubID uuid.UUID, clientRefs []ClientRef) error {
	if len(clientRefs) == 0 {
		return nil
	}

	userHubFields := make(map[uuid.UUID][]string)
	for _, ref := range clientRefs {
		field := ref.SessionID.String() + ":" + ref.ClientID.String()
		userHubFields[ref.UserID] = append(userHubFields[ref.UserID], field)
	}

	userIDs := make([]uuid.UUID, 0, len(userHubFields))
	for userID := range userHubFields {
		userIDs = append(userIDs, userID)
	}

	const batchSize = 500
	for i := 0; i < len(userIDs); i += batchSize {
		end := i + batchSize
		if end > len(userIDs) {
			end = len(userIDs)
		}

		batch := userIDs[i:end]
		pipe := c.client.Pipeline()

		for _, userID := range batch {
			removeBatchClientsScript.Run(
				ctx,
				pipe,
				[]string{UserCacheKey(userID)},
				hubID.String(),
				int(UserCacheTTL.Seconds()),
			)
		}

		if _, err := pipe.Exec(ctx); err != nil && err != redisdriver.Nil {
			return redis.NewError(err, redis.ResourceGateway)
		}
	}

	return nil
}

func (c *Cache) Heartbeat(ctx context.Context, userID uuid.UUID) error {
	if err := c.client.Expire(ctx, UserCacheKey(userID), UserCacheTTL).Err(); err != nil {
		return redis.NewError(err, redis.ResourceGateway)
	}
	return nil
}

func (c *Cache) GetPresence(ctx context.Context, userID uuid.UUID) (Presence, error) {
	val, err := c.client.HGet(ctx, UserCacheKey(userID), "presence").Result()
	if err != nil {
		if err == redisdriver.Nil {
			return PresenceOffline, nil
		}
		return PresenceOffline, redis.NewError(err, redis.ResourceGateway)
	}

	intVal, err := strconv.Atoi(val)
	if err != nil {
		return PresenceOffline, nil
	}

	presence, err := ParsePresenceInt(intVal)
	if err != nil || !presence.IsValid() {
		return PresenceOffline, nil
	}

	return presence, nil
}

func (c *Cache) SetPresence(ctx context.Context, userID uuid.UUID, p Presence) error {
	parsedPresence := p
	if !parsedPresence.IsValid() {
		parsedPresence = PresenceOnline
	}

	if err := c.client.HSet(ctx, UserCacheKey(userID), "presence", parsedPresence.Int()).Err(); err != nil {
		return redis.NewError(err, redis.ResourceGateway)
	}
	return nil
}

func (c *Cache) GetBatchPresence(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]Presence, error) {
	if len(userIDs) == 0 {
		return make(map[uuid.UUID]Presence), nil
	}

	pipe := c.client.Pipeline()
	cmds := make(map[uuid.UUID]*redisdriver.StringCmd, len(userIDs))

	for _, userID := range userIDs {
		cmds[userID] = pipe.HGet(ctx, UserCacheKey(userID), "presence")
	}

	_, err := pipe.Exec(ctx)
	if err != nil && err != redisdriver.Nil {
		return nil, redis.NewError(err, redis.ResourceGateway)
	}

	result := make(map[uuid.UUID]Presence, len(userIDs))
	for userID, cmd := range cmds {
		val, err := cmd.Result()
		if err != nil {
			result[userID] = PresenceOffline
			continue
		}

		intVal, err := strconv.Atoi(val)
		if err != nil {
			result[userID] = PresenceOffline
			continue
		}

		presence, err := ParsePresenceInt(intVal)
		if err != nil || !presence.IsValid() {
			result[userID] = PresenceOffline
			continue
		}

		result[userID] = presence
	}

	return result, nil
}
