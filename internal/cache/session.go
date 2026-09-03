package cache

import (
	"bonfire-api/internal/fields"
	"bonfire-api/internal/redis"
	"bonfire-api/internal/session"
	"context"
	"time"

	redisdriver "github.com/redis/go-redis/v9"
)

var (
	sessionTTL     = 24 * time.Hour
	sessionNodeTTL = userNodesTTL
)

const (
	sessionDomainKey = "session:"
)

func sessionNamespacedKey(id fields.ID, suffix string) string {
	if suffix == "" {
		return "{" + sessionDomainKey + id.String() + "}"
	}
	return "{" + sessionDomainKey + id.String() + "}:" + suffix
}

func sessionKey(id fields.ID) string     { return sessionNamespacedKey(id, "") }
func sessionNodeKey(id fields.ID) string { return sessionNamespacedKey(id, "node") }

type SessionCache struct {
	client redisdriver.Cmdable
}

func NewSessionCache(client redisdriver.Cmdable) *SessionCache {
	return &SessionCache{
		client: client,
	}
}

func (c *SessionCache) Get(ctx context.Context, id fields.ID) (*session.Session, error) {
	return getAndUnmarshal(ctx, c.client, sessionKey(id), redis.ScopeSession, unmarshalSession)
}

func (c *SessionCache) Set(ctx context.Context, sess *session.Session) error {
	ttl := sessionTTL
	if sess.ExpiresAt().IsValid() {
		if remaining := time.Until(sess.ExpiresAt().Time()); remaining > 0 {
			ttl = remaining
		} else {
			return nil
		}
	}
	return marshalAndSet(ctx, c.client, sessionKey(sess.ID()), sess, ttl, redis.ScopeSession, marshalSession)
}

func (c *SessionCache) Delete(ctx context.Context, id fields.ID) error {
	if err := c.client.Del(ctx, sessionKey(id)).Err(); err != nil {
		return redis.NewError(err, redis.ScopeSession)
	}
	return nil
}

func (c *SessionCache) DeleteBatch(ctx context.Context, ids []fields.ID) error {
	if len(ids) == 0 {
		return nil
	}
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = sessionKey(id)
	}
	return deleteBatchKeys(ctx, c.client, keys, redis.ScopeSession)
}
