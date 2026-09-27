package cache

import (
	"context"
	"time"

	"bonfire-api/internal/redis"

	"github.com/google/uuid"
	redisdriver "github.com/redis/go-redis/v9"
)

var (
	wsTicketTTL = 30 * time.Second
)

func wsTicketKey(ticketID uuid.UUID) string {
	return "{ws_ticket:" + ticketID.String() + "}"
}

type WSTicketCache struct {
	client redisdriver.Cmdable
}

func NewWSTicketCache(client redisdriver.Cmdable) *WSTicketCache {
	return &WSTicketCache{client: client}
}

// Print stores the ticket data mapping a ticketID to both userID and sessionID using raw bytes.
func (c *WSTicketCache) Print(ctx context.Context, ticketID, userID, sessionID uuid.UUID) error {
	var payload [32]byte
	copy(payload[0:16], userID[:])
	copy(payload[16:32], sessionID[:])

	if err := c.client.Set(ctx, wsTicketKey(ticketID), payload[:], wsTicketTTL).Err(); err != nil {
		return redis.NewError(err, redis.ScopeGateway)
	}
	return nil
}

// Punch atomically retrieves and deletes the ticket, returning both userID and sessionID.
func (c *WSTicketCache) Punch(ctx context.Context, ticketID uuid.UUID) (uuid.UUID, uuid.UUID, error) {
	val, err := c.client.GetDel(ctx, wsTicketKey(ticketID)).Bytes()
	if err != nil {
		return uuid.UUID{}, uuid.UUID{}, redis.NewError(err, redis.ScopeGateway)
	}

	if len(val) != 32 {
		return uuid.UUID{}, uuid.UUID{}, ErrInvalidWSTicketLength()
	}

	userID, _ := uuid.FromBytes(val[0:16])
	sessionID, _ := uuid.FromBytes(val[16:32])

	return userID, sessionID, nil
}
