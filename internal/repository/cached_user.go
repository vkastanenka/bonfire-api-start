package repository

import (
	"context"
	"time"

	"bonfire-api/internal/cache"
	"bonfire-api/internal/fields"
	"bonfire-api/internal/user"
)

type CachedUser struct {
	repo  *User
	cache cache.UserCache
}

func NewCachedUser(repo *User, cache cache.UserCache) *CachedUser {
	return &CachedUser{
		repo:  repo,
		cache: cache,
	}
}

func (c *CachedUser) Availability(ctx context.Context, email user.Email, username user.Username) (bool, bool, error) {
	return c.repo.Availability(ctx, email, username)
}

func (c *CachedUser) Create(ctx context.Context, u *user.User) (*user.User, error) {
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

func (c *CachedUser) Get(ctx context.Context, id fields.ID) (*user.User, error) {
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

func (c *CachedUser) GetByEmail(ctx context.Context, email user.Email) (*user.User, error) {
	// Typically lookups by secondary attributes hit the database directly
	return c.repo.GetByEmail(ctx, email)
}

func (c *CachedUser) ListDeleteScheduled(ctx context.Context, currentTime user.Timestamp, batchLimit int32) ([]*user.User, error) {
	return c.repo.ListDeleteScheduled(ctx, currentTime, batchLimit)
}

func (c *CachedUser) Update(ctx context.Context, u *user.User) (*user.User, error) {
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

func (c *CachedUser) UpdateBatch(ctx context.Context, usersJson []byte) ([]*user.User, error) {
	updatedUsers, err := c.repo.UpdateBatch(ctx, usersJson)
	if err != nil {
		return nil, err
	}

	// Extract IDs and perform batch invalidation
	ids := make([]user.ID, len(updatedUsers))
	for i, u := range updatedUsers {
		ids[i] = u.ID()
	}

	if cacheErr := c.cache.DeleteBatch(ctx, ids); cacheErr != nil {
		// Log cache error (non-fatal)
	}

	return updatedUsers, nil
}
