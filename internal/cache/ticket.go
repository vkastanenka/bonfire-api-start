package cache

import (
	"context"
	"time"

	"bonfire-api/internal/fields"
	"bonfire-api/internal/redis"

	redisdriver "github.com/redis/go-redis/v9"
)

func ticketKey(ticketID fields.ID) string {
	return "ticket:" + ticketID.String()
}

type TicketCache struct {
	client redisdriver.Cmdable
	scope  redis.Scope
	ttl    time.Duration
}

func NewTicketCache(client redisdriver.Cmdable, scope redis.Scope, ttl time.Duration) *TicketCache {
	return &TicketCache{
		client: client,
		scope:  scope,
		ttl:    ttl,
	}
}

func (c *TicketCache) Print(ctx context.Context, ticketID, userID fields.ID) error {
	if err := c.client.Set(ctx, ticketKey(ticketID), userID.String(), c.ttl).Err(); err != nil {
		return redis.NewError(err, c.scope)
	}
	return nil
}

func (c *TicketCache) Punch(ctx context.Context, ticketID fields.ID) (fields.ID, error) {
	val, err := c.client.GetDel(ctx, ticketKey(ticketID)).Result()
	if err != nil {
		return fields.ID{}, redis.NewError(err, c.scope)
	}

	userID, err := fields.ParseRequiredIDFromString("user_id", val)
	if err != nil {
		return fields.ID{}, err
	}

	return userID, nil
}
