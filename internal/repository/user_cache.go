package repository

import (
	"context"
	"log/slog"

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

	if setErr := r.cache.Set(ctx, u); setErr != nil {
		slog.WarnContext(ctx, "failed to populate user cache after database read",
			slog.String("user_id", id.String()),
			slog.Any("error", setErr),
		)
	}

	return u, nil
}

func (r *CachedUserRepository) GetBatch(
	ctx context.Context,
	ids []uuid.UUID,
) (map[uuid.UUID]*user.User, error) {
	found, missing, err := r.cache.GetBatch(ctx, ids)
	if err != nil {
		slog.WarnContext(ctx, "redis user cache batch read failure, falling back to database",
			slog.Int("requested_count", len(ids)),
			slog.Any("error", err),
		)
		missing = ids
		found = make(map[uuid.UUID]*user.User)
	}

	usersMap := make(map[uuid.UUID]*user.User, len(ids))
	for id, u := range found {
		if u != nil {
			usersMap[id] = u
		}
	}

	if len(missing) == 0 {
		return usersMap, nil
	}

	dbUsersMap, err := r.repo.GetBatch(ctx, missing)
	if err != nil {
		return nil, err
	}

	if len(dbUsersMap) > 0 {
		if setErr := r.cache.SetBatch(ctx, dbUsersMap); setErr != nil {
			slog.WarnContext(ctx, "failed to populate user batch cache after database read",
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
