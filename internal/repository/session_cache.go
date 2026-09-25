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

func (r *CachedSessionRepository) ListValidByUserID(
	ctx context.Context,
	userID uuid.UUID,
	now time.Time,
	limit int,
) ([]*session.Session, error) {
	// Fetch indexed session IDs for the user from cache
	sessionIDs, err := r.cache.GetUserSessionIDs(ctx, userID)
	if err != nil {
		slog.WarnContext(ctx, "redis user sessions index read failure, falling back to database",
			slog.String("user_id", userID.String()),
			slog.Any("error", err),
		)
	}

	// Cache Index Hit -> Attempt to fetch individual sessions from cache
	if len(sessionIDs) > 0 {
		found, missing, err := r.cache.GetBatch(ctx, sessionIDs)
		if err != nil {
			slog.WarnContext(ctx, "redis session batch read failure, falling back to database",
				slog.String("user_id", userID.String()),
				slog.Any("error", err),
			)
		} else if len(missing) == 0 {
			// Complete cache hit: Filter out expired ones and apply limit
			sessions := make([]*session.Session, 0, len(found))
			for _, s := range found {
				if s != nil && s.IsValid(now) {
					sessions = append(sessions, s)
				}
			}

			if limit > 0 && len(sessions) > limit {
				return sessions[:limit], nil
			}
			return sessions, nil
		}
	}

	// Cache Miss (or Partial Miss) -> Fall back to DB
	dbSessions, err := r.repo.ListValidByUserID(ctx, userID, now, limit)
	if err != nil {
		return nil, err
	}

	cacheCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()

	// Backfill Redis Cache (both payloads and set index)
	if len(dbSessions) > 0 {
		sessionMap := make(map[uuid.UUID]*session.Session, len(dbSessions))
		fetchedIDs := make([]uuid.UUID, 0, len(dbSessions))

		for _, s := range dbSessions {
			sessionMap[s.ID] = s
			fetchedIDs = append(fetchedIDs, s.ID)
		}

		if setErr := r.cache.SetBatch(cacheCtx, sessionMap); setErr != nil {
			slog.WarnContext(cacheCtx, "failed to backfill session batch cache after database read",
				slog.String("user_id", userID.String()),
				slog.Int("count", len(sessionMap)),
				slog.Any("error", setErr),
			)
		}

		if setErr := r.cache.SetUserSessionIDs(cacheCtx, userID, fetchedIDs); setErr != nil {
			slog.WarnContext(cacheCtx, "failed to backfill user session index cache after database read",
				slog.String("user_id", userID.String()),
				slog.Int("count", len(fetchedIDs)),
				slog.Any("error", setErr),
			)
		}
	} else {
		// User has zero active sessions in DB, clear any leftover index key in Redis
		if delErr := r.cache.DeleteUserSessionsIndex(cacheCtx, userID); delErr != nil {
			slog.WarnContext(cacheCtx, "failed to clear empty user session index cache",
				slog.String("user_id", userID.String()),
				slog.Any("error", delErr),
			)
		}
	}

	return dbSessions, nil
}
