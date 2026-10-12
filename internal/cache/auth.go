package cache

import (
	"bonfire-api/internal/redis"
	"bonfire-api/internal/session"
	"bonfire-api/internal/user"
	"context"

	redisdriver "github.com/redis/go-redis/v9"
)

type AuthCache struct {
	rdb redisdriver.Cmdable
}

func NewAuthCache(rdb redisdriver.Cmdable) *AuthCache {
	return &AuthCache{rdb: rdb}
}

func (s *AuthCache) Login(ctx context.Context, u *user.User, sess *session.Session) error {
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

	_, err = s.rdb.Pipelined(ctx, func(pipe redisdriver.Pipeliner) error {
		pipe.Set(ctx, userKey, userBytes, user.CacheTTL)
		pipe.Set(ctx, sessKey, sessBytes, sessTTL)
		return nil
	})
	if err != nil {
		return redis.NewError(err, redis.ScopeAuth)
	}

	return nil
}
