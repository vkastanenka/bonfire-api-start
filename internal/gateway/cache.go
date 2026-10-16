package gateway

import (
	"context"
	"time"

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

// PrintTicket stores the ticket data mapping a ticketID to userID, sessionID, and clientID using raw bytes.
func (c *Cache) PrintTicket(ctx context.Context, ticketID, userID, sessionID, clientID uuid.UUID) error {
	var payload [48]byte
	copy(payload[0:16], userID[:])
	copy(payload[16:32], sessionID[:])
	copy(payload[32:48], clientID[:]) // TODO: Add here or in the client initializer?

	if err := c.client.Set(ctx, TicketCacheKey(ticketID), payload[:], TicketCacheTTL).Err(); err != nil {
		return redis.NewError(err, redis.ResourceGateway)
	}
	return nil
}

// PunchTicket atomically retrieves and deletes the ticket, returning both userID and sessionID.
func (c *Cache) PunchTicket(ctx context.Context, ticketID uuid.UUID) (uuid.UUID, uuid.UUID, uuid.UUID, error) {
	val, err := c.client.GetDel(ctx, TicketCacheKey(ticketID)).Bytes()
	if err != nil {
		return uuid.UUID{}, uuid.UUID{}, uuid.UUID{}, redis.NewError(err, redis.ResourceGateway)
	}

	if len(val) != 48 {
		return uuid.UUID{}, uuid.UUID{}, uuid.UUID{}, ErrInvalidWSTicketLength()
	}

	userID := uuid.Must(uuid.FromBytes(val[0:16]))
	sessionID := uuid.Must(uuid.FromBytes(val[16:32]))
	clientID := uuid.Must(uuid.FromBytes(val[32:48]))

	return userID, sessionID, clientID, nil
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

// GetBatchUsers retrieves the raw Redis hash fields for a batch of user IDs.
// Returns a map of userID -> hash fields.
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
