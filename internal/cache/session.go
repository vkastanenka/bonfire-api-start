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
	defaultSessionTTL = 24 * time.Hour
	minEffectiveTTL   = 1 * time.Second
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
	if sess == nil {
		return nil
	}

	ttl, ok := calcEffSessionTTL(sess)
	if !ok {
		return c.Delete(ctx, sess.ID)
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
	return getAndUnmarshalBatch(ctx, redis.ScopeSession, c.client, ids, sessionKey, unmarshalSession)
}

func (c *SessionCache) SetBatch(ctx context.Context, sessions map[uuid.UUID]*session.Session) error {
	if len(sessions) == 0 {
		return nil
	}

	sessionList := make([]*session.Session, 0, len(sessions))
	for _, sess := range sessions {
		if sess != nil {
			sessionList = append(sessionList, sess)
		}
	}

	for i := 0; i < len(sessionList); i += maxBatchSize {
		if err := ctx.Err(); err != nil {
			return err
		}

		end := min(i+maxBatchSize, len(sessionList))
		chunk := sessionList[i:end]

		pipe := c.client.Pipeline()
		var queued int

		for _, sess := range chunk {
			ttl, ok := calcEffSessionTTL(sess)
			key := sessionKey(sess.ID)

			if !ok {
				pipe.Del(ctx, key)
				queued++
				continue
			}

			data, err := marshalSession(sess)
			if err != nil {
				return redis.NewError(err, redis.ScopeSession)
			}

			pipe.Set(ctx, key, data, ttl)
			queued++
		}

		if queued > 0 {
			if _, err := pipe.Exec(ctx); err != nil {
				return redis.NewError(err, redis.ScopeSession)
			}
		}
	}

	return nil
}

func (c *SessionCache) DeleteBatch(ctx context.Context, ids []uuid.UUID) error {
	return deleteBatch(ctx, redis.ScopeSession, c.client, ids, sessionKey)
}

func calcEffSessionTTL(sess *session.Session) (time.Duration, bool) {
	if sess.ExpiresAt.IsZero() {
		return defaultSessionTTL, true
	}

	remaining := time.Until(sess.ExpiresAt)
	if remaining < minEffectiveTTL {
		return 0, false
	}

	return remaining, true
}
