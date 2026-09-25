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

func sessionKey(id uuid.UUID) string { return "{session:" + id.String() + "}" }

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

func (c *SessionCache) GetBatch(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*session.Session, []uuid.UUID, error) {
	return getAndUnmarshalBatch(ctx, redis.ScopeUser, c.client, ids, sessionKey, unmarshalSession)
}

func (c *SessionCache) SetBatch(ctx context.Context, users map[uuid.UUID]*session.Session) error {
	return marshalAndSetBatch(ctx, redis.ScopeUser, c.client, users, sessionKey, sessionTTL, marshalSession)
}

func (c *SessionCache) DeleteBatch(ctx context.Context, ids []uuid.UUID) error {
	return deleteBatch(ctx, redis.ScopeSession, c.client, ids, sessionKey)
}

func (c *SessionCache) GetUserSessionIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	return getSetIDs(ctx, redis.ScopeSession, c.client, userSessionsKey(userID))
}

func (c *SessionCache) SetUserSessionIDs(ctx context.Context, userID uuid.UUID, sessionIDs []uuid.UUID) error {
	return setSetIDs(ctx, redis.ScopeSession, c.client, userSessionsKey(userID), sessionIDs, sessionTTL)
}

func (c *SessionCache) AddUserSessionID(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID) error {
	return addToSetIDs(ctx, redis.ScopeSession, c.client, userSessionsKey(userID), sessionTTL, sessionID)
}

func (c *SessionCache) RemoveUserSessionID(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID) error {
	return removeFromSetIDs(ctx, redis.ScopeSession, c.client, userSessionsKey(userID), sessionID)
}

func (c *SessionCache) DeleteUserSessionsIndex(ctx context.Context, userID uuid.UUID) error {
	return deleteSet(ctx, redis.ScopeSession, c.client, userSessionsKey(userID))
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
