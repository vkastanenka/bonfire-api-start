package repository

import (
	"context"

	"bonfire-api/internal/fields"
	"bonfire-api/internal/user"

	"github.com/google/uuid"
)

type CachedUserRepository struct {
	cache UserCache
	repo  UserRepository
}

func NewCachedUserRepository(cache UserCache, repo UserRepository) *CachedUserRepository {
	return &CachedUserRepository{
		cache: cache,
		repo:  repo,
	}
}

func (r *CachedUserRepository) Get(ctx context.Context, id uuid.UUID) (*user.User, error) {
	u, err := r.cache.Get(ctx, id)
	if err == nil && u != nil {
		return u, nil
	}

	u, err = r.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	_ = r.cache.Set(ctx, u)

	return u, nil
}

func (r *CachedUserRepository) GetBatch(
	ctx context.Context,
	ids []uuid.UUID,
) (map[uuid.UUID]*user.User, error) {
	if len(ids) == 0 {
		return make(map[uuid.UUID]*user.User), nil
	}

	found, missing, err := r.cache.GetBatch(ctx, ids)
	if err != nil {
		missing = ids
		found = make(map[uuid.UUID]*user.User)
	}

	usersMap := make(map[uuid.UUID]*user.User, len(ids))
	for id, u := range found {
		if u != nil {
			usersMap[id] = u
		} else {
			missing = append(missing, id)
		}
	}

	if len(missing) == 0 {
		return usersMap, nil
	}

	missing = fields.DedupeIDs(missing)

	dbUsersMap, err := r.repo.GetBatch(ctx, missing)
	if err != nil {
		return nil, err
	}

	if len(dbUsersMap) > 0 {
		_ = r.cache.SetBatch(ctx, dbUsersMap)
		for id, u := range dbUsersMap {
			usersMap[id] = u
		}
	}

	return usersMap, nil
}
