package auth

import (
	"bonfire-api/internal/pkg/shield"
	"bonfire-api/internal/redis"
	"bonfire-api/internal/user"
	"context"

	"github.com/google/uuid"
	redisdriver "github.com/redis/go-redis/v9"
)

type Cache struct {
	rdb redisdriver.Cmdable
}

func NewCache(rdb redisdriver.Cmdable) *Cache {
	return &Cache{rdb: rdb}
}

func (s *Cache) Login(ctx context.Context, u *user.User, sess *Session) error {
	return s.commitSession(ctx, u, sess, nil)
}

func (s *Cache) ResetPassword(ctx context.Context, u *user.User, sess *Session, revokedSessionIDs []uuid.UUID) error {
	return s.commitSession(ctx, u, sess, revokedSessionIDs)
}

func (s *Cache) commitSession(
	ctx context.Context,
	u *user.User,
	sess *Session,
	revokedSessionIDs []uuid.UUID,
) error {
	userKey := user.CacheKey(u.ID)
	sessKey := SessionCacheKey(sess.ID)

	sessTTL, ok := CalcSessionCacheTTL(sess)
	if !ok {
		return nil
	}

	userBytes, err := user.MarshalCacheDTO(u)
	if err != nil {
		return err
	}

	sessBytes, err := MarshalSessionCacheDTO(sess)
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

		if len(revokedSessionIDs) > 0 {
			keys := make([]string, len(revokedSessionIDs))
			for i, id := range revokedSessionIDs {
				keys[i] = SessionCacheKey(id)
			}
			pipe.Del(ctx, keys...)
		}

		return nil
	})
	if err != nil {
		return redis.NewError(err, redis.ResourceAuth)
	}

	return nil
}
