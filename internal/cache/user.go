package cache

import (
	"context"
	"time"

	"bonfire-api/internal/redis"
	"bonfire-api/internal/user"

	"github.com/google/uuid"
	redisdriver "github.com/redis/go-redis/v9"
)

var (
	userTTL = 24 * time.Hour
)

func userKey(id uuid.UUID) string { return "{user:" + id.String() + "}" }

type UserCache struct {
	client redisdriver.Cmdable
}

func NewUserCache(client redisdriver.Cmdable) *UserCache {
	return &UserCache{client: client}
}

func (c *UserCache) Get(ctx context.Context, id uuid.UUID) (*user.User, error) {
	return getAndUnmarshal(ctx, redis.ScopeUser, c.client, userKey(id), unmarshalUser)
}

func (c *UserCache) Set(ctx context.Context, usr *user.User) error {
	return marshalAndSet(ctx, redis.ScopeUser, c.client, userKey(usr.ID), usr, userTTL, marshalUser)
}

func (c *UserCache) Delete(ctx context.Context, id uuid.UUID) error {
	if err := c.client.Del(ctx, userKey(id)).Err(); err != nil {
		return redis.NewError(err, redis.ScopeUser)
	}
	return nil
}

func (c *UserCache) GetBatch(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*user.User, []uuid.UUID, error) {
	return getAndUnmarshalBatch(ctx, redis.ScopeUser, c.client, ids, userKey, unmarshalUser)
}

func (c *UserCache) SetBatch(ctx context.Context, users map[uuid.UUID]*user.User) error {
	return marshalAndSetBatch(ctx, redis.ScopeUser, c.client, users, userKey, userTTL, marshalUser)
}

func (c *UserCache) DeleteBatch(ctx context.Context, ids []uuid.UUID) error {
	return deleteBatch(ctx, redis.ScopeUser, c.client, ids, userKey)
}
