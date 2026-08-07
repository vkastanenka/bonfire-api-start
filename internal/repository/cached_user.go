package repository

import (
	"context"
	"time"

	"bonfire-api/internal/fields"
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

	// Warm the cache on creation
	if cacheErr := c.cache.Set(ctx, createdUser); cacheErr != nil {
		// Log cache error (non-fatal)
	}

	return createdUser, nil
}

func (c *CachedUserRepository) Get(ctx context.Context, id fields.ID) (*user.User, error) {
	// 1. Try reading from cache first
	u, err := c.cache.Get(ctx, id)
	if err == nil && u != nil {
		return u, nil
	}

	// 2. Fall back to database on cache miss/error
	u, err = c.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	// 3. Asynchronously backfill the cache
	go func(userToCache *user.User) {
		// Use a detached context with a tight timeout to prevent leaking requests
		asyncCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
		defer cancel()

		if cacheErr := c.cache.Set(asyncCtx, userToCache); cacheErr != nil {
			// Log cache error
		}
	}(u)

	return u, nil
}

func (c *CachedUserRepository) Update(ctx context.Context, u *user.User) (*user.User, error) {
	updatedUser, err := c.repo.Update(ctx, u)
	if err != nil {
		return nil, err
	}

	// Invalidate cache entry on modification
	if cacheErr := c.cache.Delete(ctx, u.ID()); cacheErr != nil {
		// Log cache error (non-fatal)
	}

	return updatedUser, nil
}

func (c *CachedUserRepository) UpdateBatch(ctx context.Context, usersJson []byte) ([]*user.User, error) {
	updatedUsers, err := c.repo.UpdateBatch(ctx, usersJson)
	if err != nil {
		return nil, err
	}

	// Extract IDs and perform batch invalidation
	ids := make([]fields.ID, len(updatedUsers))
	for i, u := range updatedUsers {
		ids[i] = u.ID()
	}

	if cacheErr := c.cache.DeleteBatch(ctx, ids); cacheErr != nil {
		// Log cache error (non-fatal)
	}

	return updatedUsers, nil
}
