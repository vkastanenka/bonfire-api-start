package cache

import (
	"context"
	"encoding/json"
	"time"

	"bonfire-api/internal/fields"
	"bonfire-api/internal/redis"

	redisdriver "github.com/redis/go-redis/v9"
)

func wsTicketKey(ticketID fields.ID) string {
	return "ticket:ws:" + ticketID.String()
}

type WSTicketData struct {
	UserID    fields.ID `json:"user_id"`
	SessionID fields.ID `json:"session_id"`
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
func (c *WSTicketCache) Print(ctx context.Context, ticketID, userID, sessionID fields.ID) error {
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
func (c *WSTicketCache) Punch(ctx context.Context, ticketID fields.ID) (fields.ID, fields.ID, error) {
	val, err := c.client.GetDel(ctx, wsTicketKey(ticketID)).Result()
	if err != nil {
		return fields.ID{}, fields.ID{}, redis.NewError(err, c.scope)
	}

	var data WSTicketData
	if err := json.Unmarshal([]byte(val), &data); err != nil {
		return fields.ID{}, fields.ID{}, redis.NewError(err, c.scope)
	}

	return data.UserID, data.SessionID, nil
}
