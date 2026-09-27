package repository

import (
	"bonfire-api/internal/session"
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

type CachedSessionRepository struct {
	cache SessionCache
	repo  *SessionRepository
}

func NewCachedSessionRepository(cache SessionCache, repo *SessionRepository) *CachedSessionRepository {
	return &CachedSessionRepository{cache: cache, repo: repo}
}

func (r *CachedSessionRepository) Get(ctx context.Context, id uuid.UUID) (*session.Session, error) {
	s, err := r.cache.Get(ctx, id)
	if err == nil && s != nil {
		return s, nil
	}

	if err != nil {
		slog.WarnContext(ctx, "redis session cache read failure, falling back to database",
			slog.String("session_id", id.String()),
			slog.Any("error", err),
		)
	}

	s, err = r.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	cacheCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()

	if setErr := r.cache.Set(cacheCtx, s); setErr != nil {
		slog.WarnContext(cacheCtx, "failed to populate session cache after database read",
			slog.String("session_id", id.String()),
			slog.Any("error", setErr),
		)
	}

	return s, nil
}
