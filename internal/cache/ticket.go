package cache

import (
	"context"
	"encoding/json"
	"time"

	"bonfire-api/internal/redis"

	"github.com/google/uuid"
	redisdriver "github.com/redis/go-redis/v9"
)

func wsTicketKey(ticketID uuid.UUID) string {
	return "ticket:ws:" + ticketID.String()
}

type WSTicketData struct {
	UserID    uuid.UUID `json:"user_id"`
	SessionID uuid.UUID `json:"session_id"`
}

type WSTicketCache struct {
	client redisdriver.Cmdable
	scope  redis.Scope
	ttl    time.Duration
}

func NewWSTicketCache(client redisdriver.Cmdable, scope redis.Scope, ttl time.Duration) *WSTicketCache {
	return &WSTicketCache{
		client: client,
		scope:  scope,
		ttl:    ttl,
	}
}

// Print stores the ticket data mapping a ticketID to both userID and sessionID.
func (c *WSTicketCache) Print(ctx context.Context, ticketID, userID, sessionID uuid.UUID) error {
	data := WSTicketData{
		UserID:    userID,
		SessionID: sessionID,
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return redis.NewError(err, c.scope)
	}

	if err := c.client.Set(ctx, wsTicketKey(ticketID), payload, c.ttl).Err(); err != nil {
		return redis.NewError(err, c.scope)
	}
	return nil
}

// Punch atomically retrieves and deletes the ticket, returning both userID and sessionID.
func (c *WSTicketCache) Punch(ctx context.Context, ticketID uuid.UUID) (uuid.UUID, uuid.UUID, error) {
	val, err := c.client.GetDel(ctx, wsTicketKey(ticketID)).Result()
	if err != nil {
		return uuid.UUID{}, uuid.UUID{}, redis.NewError(err, c.scope)
	}

	var data WSTicketData
	if err := json.Unmarshal([]byte(val), &data); err != nil {
		return uuid.UUID{}, uuid.UUID{}, redis.NewError(err, c.scope)
	}

	return data.UserID, data.SessionID, nil
}
