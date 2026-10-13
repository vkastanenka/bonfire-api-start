package user

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

type CachedRepository struct {
	cache *Cache
	repo  *Repository
}

func NewCachedRepository(cache *Cache, repo *Repository) *CachedRepository {
	return &CachedRepository{cache: cache, repo: repo}
}

func (r *CachedRepository) Get(ctx context.Context, id uuid.UUID) (*User, error) {
	u, err := r.cache.Get(ctx, id)
	if err == nil && u != nil {
		return u, nil
	}

	if err != nil {
		slog.WarnContext(ctx, "redis user cache read failure, falling back to database",
			slog.String("user_id", id.String()),
			slog.Any("error", err),
		)
	}

	u, err = r.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	cacheCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()

	if setErr := r.cache.Set(cacheCtx, u); setErr != nil {
		slog.WarnContext(cacheCtx, "failed to populate user cache after database read",
			slog.String("user_id", id.String()),
			slog.Any("error", setErr),
		)
	}

	return u, nil
}

func (r *CachedRepository) GetBatch(
	ctx context.Context,
	ids []uuid.UUID,
) (map[uuid.UUID]*User, error) {
	found, missing, err := r.cache.GetBatch(ctx, ids)
	if err != nil {
		slog.WarnContext(ctx, "redis user cache batch read failure, falling back to database",
			slog.Int("requested_count", len(ids)),
			slog.Any("error", err),
		)
		missing = ids
		found = make(map[uuid.UUID]*User)
	}

	usersMap := make(map[uuid.UUID]*User, len(ids))
	for id, u := range found {
		if u != nil {
			usersMap[id] = u
		}
	}

	if len(missing) == 0 {
		return usersMap, nil
	}

	dbUsers, err := r.repo.GetBatch(ctx, missing)
	if err != nil {
		return nil, err
	}

	dbUsersMap := make(map[uuid.UUID]*User, len(dbUsers))
	for _, u := range dbUsers {
		if u != nil {
			dbUsersMap[u.ID] = u
		}
	}

	cacheCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()

	if len(dbUsersMap) > 0 {
		if setErr := r.cache.SetBatch(cacheCtx, dbUsersMap); setErr != nil {
			slog.WarnContext(cacheCtx, "failed to populate user batch cache after database read",
				slog.Int("count", len(dbUsersMap)),
				slog.Any("error", setErr),
			)
		}

		for id, u := range dbUsersMap {
			usersMap[id] = u
		}
	}

	return usersMap, nil
}
