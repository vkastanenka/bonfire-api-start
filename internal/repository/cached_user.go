package repository

import (
	"context"
	"errors"
	"log/slog"

	"bonfire-api/internal/fields"
	"bonfire-api/internal/redis"
	"bonfire-api/internal/user"
)

type userCache interface {
	Delete(ctx context.Context, userID fields.ID) error
	DeleteBatch(ctx context.Context, userIDs []fields.ID) error
	Get(ctx context.Context, userID fields.ID) (*user.User, error)
	Set(ctx context.Context, usr *user.User) error
}

type userRepository interface {
	Create(ctx context.Context, u *user.User) (*user.User, error)
	Get(ctx context.Context, id fields.ID) (*user.User, error)
	Update(ctx context.Context, u *user.User) (*user.User, error)
	UpdateBatch(ctx context.Context, usersJson []byte) ([]*user.User, error)
}

type CachedUserRepository struct {
	repo  userRepository
	cache userCache
}

func NewCachedUser(repo *UserRepository, cache userCache) *CachedUserRepository {
	return &CachedUserRepository{
		repo:  repo,
		cache: cache,
	}
}

func (c *CachedUserRepository) Create(ctx context.Context, u *user.User) (*user.User, error) {
	createdUser, err := c.repo.Create(ctx, u)
	if err != nil {
		return nil, err
	}

	if cacheErr := c.cache.Set(ctx, createdUser); cacheErr != nil {
		slog.WarnContext(ctx, "failed to cache newly created user",
			"user_id", createdUser.ID().String(),
			"error", cacheErr,
			"scope", redis.ScopeUser,
		)
	}

	return createdUser, nil
}

func (c *CachedUserRepository) Get(ctx context.Context, id fields.ID) (*user.User, error) {
	u, err := c.cache.Get(ctx, id)
	if err == nil && u != nil {
		return u, nil
	}
	if err != nil && !errors.Is(err, redis.ErrCacheMiss) {
		slog.WarnContext(ctx, "user cache read failed, falling back to database",
			"user_id", id.String(),
			"error", err,
			"scope", redis.ScopeUser,
		)
	}

	u, err = c.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if cacheErr := c.cache.Set(ctx, u); cacheErr != nil {
		slog.WarnContext(ctx, "failed to backfill user cache",
			"user_id", u.ID().String(),
			"error", cacheErr,
			"scope", redis.ScopeUser,
		)
	}

	return u, nil
}

func (c *CachedUserRepository) Update(ctx context.Context, u *user.User) (*user.User, error) {
	updatedUser, err := c.repo.Update(ctx, u)
	if err != nil {
		return nil, err
	}

	if cacheErr := c.cache.Delete(ctx, u.ID()); cacheErr != nil {
		slog.WarnContext(ctx, "failed to invalidate user cache after update",
			"user_id", u.ID().String(),
			"error", cacheErr,
			"scope", redis.ScopeUser,
		)
	}

	return updatedUser, nil
}

func (c *CachedUserRepository) UpdateBatch(ctx context.Context, usersJson []byte) ([]*user.User, error) {
	updatedUsers, err := c.repo.UpdateBatch(ctx, usersJson)
	if err != nil {
		return nil, err
	}

	if len(updatedUsers) == 0 {
		return updatedUsers, nil
	}

	ids := make([]fields.ID, len(updatedUsers))
	for i, u := range updatedUsers {
		ids[i] = u.ID()
	}

	if cacheErr := c.cache.DeleteBatch(ctx, ids); cacheErr != nil {
		slog.WarnContext(ctx, "failed to batch invalidate user cache after update",
			"count", len(ids),
			"error", cacheErr,
			"scope", redis.ScopeUser,
		)
	}

	return updatedUsers, nil
}
