package cache

import (
	"bonfire-api/internal/pkg/shield"
	"bonfire-api/internal/redis"
	"bonfire-api/internal/session"
	"bonfire-api/internal/user"
	"context"

	"github.com/google/uuid"
	redisdriver "github.com/redis/go-redis/v9"
)

type AuthCache struct {
	rdb redisdriver.Cmdable
}

func NewAuthCache(rdb redisdriver.Cmdable) *AuthCache {
	return &AuthCache{rdb: rdb}
}

func (s *AuthCache) Login(ctx context.Context, u *user.User, sess *session.Session) error {
	return s.commitAuthSession(ctx, u, sess, nil)
}

func (s *AuthCache) ResetPassword(ctx context.Context, u *user.User, sess *session.Session, revokedSessionIDs []uuid.UUID) error {
	return s.commitAuthSession(ctx, u, sess, func(pipe redisdriver.Pipeliner) error {
		if len(revokedSessionIDs) > 0 {
			keys := make([]string, len(revokedSessionIDs))
			for i, id := range revokedSessionIDs {
				keys[i] = session.CacheKey(id)
			}
			pipe.Del(ctx, keys...)
		}
		return nil
	})
}

func (s *AuthCache) commitAuthSession(
	ctx context.Context,
	u *user.User,
	sess *session.Session,
	extraOps func(pipe redisdriver.Pipeliner) error,
) error {
	userKey := user.CacheKey(u.ID)
	sessKey := session.CacheKey(sess.ID)

	sessTTL, ok := session.CalcCacheTTL(sess)
	if !ok {
		return nil
	}

	userBytes, err := marshalUser(u)
	if err != nil {
		return err
	}

	sessBytes, err := marshalSession(sess)
	if err != nil {
		return err
	}

	loginCooldownKey := shield.CooldownKey("auth", "login", u.Email)
	loginFailuresKey := shield.FailuresKey("auth", "login", u.Email)

	_, err = s.rdb.Pipelined(ctx, func(pipe redisdriver.Pipeliner) error {
		pipe.Set(ctx, userKey, userBytes, user.CacheTTL)
		pipe.Set(ctx, sessKey, sessBytes, sessTTL)
		pipe.Del(ctx, loginCooldownKey)
		pipe.Del(ctx, loginFailuresKey)

		if extraOps != nil {
			return extraOps(pipe)
		}

		return nil
	})
	if err != nil {
		return redis.NewError(err, redis.ResourceAuth)
	}

	return nil
}
