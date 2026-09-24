package cache

import (
	"bonfire-api/internal/redis"
	"bonfire-api/internal/session"
	"context"
	"time"

	"github.com/google/uuid"
	redisdriver "github.com/redis/go-redis/v9"
)

var (
	sessionTTL = 24 * time.Hour
)

const (
	sessionDomainKey = "session:"
)

func sessionNamespacedKey(id uuid.UUID, suffix string) string {
	if suffix == "" {
		return "{" + sessionDomainKey + id.String() + "}"
	}
	return "{" + sessionDomainKey + id.String() + "}:" + suffix
}

func sessionKey(id uuid.UUID) string { return sessionNamespacedKey(id, "") }

type SessionCache struct {
	client redisdriver.Cmdable
}

func NewSessionCache(client redisdriver.Cmdable) *SessionCache {
	return &SessionCache{
		client: client,
	}
}

func (c *SessionCache) Get(ctx context.Context, id uuid.UUID) (*session.Session, error) {
	return getAndUnmarshal(ctx, redis.ScopeSession, c.client, sessionKey(id), unmarshalSession)
}

func (c *SessionCache) Set(ctx context.Context, sess *session.Session) error {
	ttl, ok := calcEffSessionTTL(sess)
	if !ok {
		return nil
	}
	return marshalAndSet(ctx, redis.ScopeSession, c.client, sessionKey(sess.ID), sess, ttl, marshalSession)
}

func (c *SessionCache) Delete(ctx context.Context, id uuid.UUID) error {
	if err := c.client.Del(ctx, sessionKey(id)).Err(); err != nil {
		return redis.NewError(err, redis.ScopeSession)
	}
	return nil
}

func (c *SessionCache) DeleteBatch(ctx context.Context, ids []uuid.UUID) error {
	return deleteBatch(ctx, redis.ScopeSession, c.client, ids, sessionKey)
}

func calcEffSessionTTL(sess *session.Session) (time.Duration, bool) {
	if sess.ExpiresAt.IsZero() {
		return sessionTTL, true
	}

	remaining := time.Until(sess.ExpiresAt)
	if remaining <= 0 {
		return 0, false
	}

	return remaining, true
}
