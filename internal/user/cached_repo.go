package user

import (
	"context"

	"golang.org/x/sync/singleflight"
)

type CachedRepository struct {
	repo  Repository
	cache Cache
	sf    singleflight.Group
}

func NewCachedRepository(repo Repository, cache Cache) *CachedRepository {
	return &CachedRepository{
		repo:  repo,
		cache: cache,
	}
}

func (r *CachedRepository) Get(ctx context.Context, id ID) (*User, error) {
	// 1. Hit Cache
	u, err := r.cache.Get(ctx, id)
	if err == nil && u != nil {
		return u, nil
	}

	// 2. Singleflight prevents Cache Stampede on DB
	v, err, _ := r.sf.Do(id.String(), func() (interface{}, error) {
		dbUser, dbErr := r.repo.Get(ctx, id)
		if dbErr != nil {
			return nil, dbErr
		}

		// Synchronous or worker-pooled cache set
		_ = r.cache.Set(ctx, dbUser)
		return dbUser, nil
	})

	if err != nil {
		return nil, err
	}

	return v.(*User), nil
}

func (r *CachedRepository) Update(ctx context.Context, u *User) (*User, error) {
	// Invalidate cache prior to write or rely on transactional outbox events
	_ = r.cache.Delete(ctx, u.ID())

	updatedUser, err := r.repo.Update(ctx, u)
	if err != nil {
		return nil, err
	}

	// Double-delete to mitigate concurrent read race conditions
	_ = r.cache.Delete(ctx, u.ID())

	return updatedUser, nil
}
