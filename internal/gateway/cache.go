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
)

func TicketCacheKey(ticketID uuid.UUID) string {
	return "gateway:ticket:" + ticketID.String()
}

type Cache struct {
	client redisdriver.Cmdable
}

func NewCache(client redisdriver.Cmdable) *Cache {
	return &Cache{client: client}
}

// PrintTicket stores the ticket data mapping a ticketID to both userID and sessionID using raw bytes.
func (c *Cache) PrintTicket(ctx context.Context, ticketID, userID, sessionID uuid.UUID) error {
	var payload [32]byte
	copy(payload[0:16], userID[:])
	copy(payload[16:32], sessionID[:])

	if err := c.client.Set(ctx, TicketCacheKey(ticketID), payload[:], TicketCacheTTL).Err(); err != nil {
		return redis.NewError(err, redis.ResourceGateway)
	}
	return nil
}

// PunchTicket atomically retrieves and deletes the ticket, returning both userID and sessionID.
func (c *Cache) PunchTicket(ctx context.Context, ticketID uuid.UUID) (uuid.UUID, uuid.UUID, error) {
	val, err := c.client.GetDel(ctx, TicketCacheKey(ticketID)).Bytes()
	if err != nil {
		return uuid.UUID{}, uuid.UUID{}, redis.NewError(err, redis.ResourceGateway)
	}

	if len(val) != 32 {
		return uuid.UUID{}, uuid.UUID{}, ErrInvalidWSTicketLength()
	}

	userID, _ := uuid.FromBytes(val[0:16])
	sessionID, _ := uuid.FromBytes(val[16:32])

	return userID, sessionID, nil
}
