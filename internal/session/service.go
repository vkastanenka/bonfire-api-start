package session

import (
	"bonfire-api/internal/appctx"
	"bonfire-api/internal/outbox"
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

type Service struct {
	cache      Cache
	repo       Repository
	cachedRepo CachedRepository
	outboxRepo OutboxRepository
	tx         TX
}

func NewService(
	cache Cache,
	repo Repository,
	cachedRepo CachedRepository,
	outboxRepo OutboxRepository,
	tx TX,
) *Service {
	return &Service{
		cache:      cache,
		repo:       repo,
		cachedRepo: cachedRepo,
		outboxRepo: outboxRepo,
		tx:         tx,
	}
}

func (s *Service) ListValidByUserID(ctx context.Context) ([]*Session, error) {
	claims, err := appctx.GetClaims(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now()

	sessions, err := s.cachedRepo.ListValidByUserID(ctx, claims.UserID, now, listValidByUserIDLimit)
	if err != nil {
		return nil, err
	}

	sort(sessions)

	return sessions, nil
}

type RevokeParams struct {
	SessionID uuid.UUID
}

func (s *Service) Revoke(ctx context.Context, p RevokeParams) error {
	claims, err := appctx.GetClaims(ctx)
	if err != nil {
		return err
	}

	now := time.Now()

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		if err := s.repo.Revoke(txCtx, p.SessionID, claims.UserID, now); err != nil {
			return err
		}

		payload := EventRevokedPayload{
			UserID:    claims.UserID,
			SessionID: p.SessionID,
			RevokedAt: now,
		}

		event, err := outbox.New(
			claims.UserID,
			claims.SessionID,
			appctx.GetTraceID(ctx),
			[]uuid.UUID{},
			EventRevoked,
			payload,
			now,
		)
		if err != nil {
			return err
		}

		return s.outboxRepo.Create(txCtx, event)
	})
	if err != nil {
		return err
	}

	s.invalidateSingleSessionCache(ctx, claims.UserID, p.SessionID, "revoke session")

	return nil
}

func (s *Service) RevokeAll(ctx context.Context) error {
	claims, err := appctx.GetClaims(ctx)
	if err != nil {
		return err
	}

	now := time.Now()

	activeSessions, err := s.cachedRepo.ListValidByUserID(ctx, claims.UserID, now, listValidByUserIDLimit)
	if err != nil {
		return err
	}

	sessionIDs := make([]uuid.UUID, len(activeSessions))
	for i, sess := range activeSessions {
		sessionIDs[i] = sess.ID
	}

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		if _, err := s.repo.RevokeAll(txCtx, claims.UserID, now); err != nil {
			return err
		}

		payload := EventRevokedAllPayload{
			UserID:     claims.UserID,
			SessionIDs: sessionIDs,
			RevokedAt:  now,
		}

		event, err := outbox.New(
			claims.UserID,
			claims.SessionID,
			appctx.GetTraceID(ctx),
			[]uuid.UUID{},
			EventRevoked,
			payload,
			now,
		)
		if err != nil {
			return err
		}

		return s.outboxRepo.Create(txCtx, event)
	})
	if err != nil {
		return err
	}

	s.invalidateAllSessionsCache(ctx, claims.UserID, sessionIDs, "revoke all sessions")

	return nil
}

func (s *Service) invalidateSingleSessionCache(ctx context.Context, userID, sessionID uuid.UUID, action string) {
	cacheCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()

	if err := s.cache.Delete(cacheCtx, sessionID); err != nil {
		slog.WarnContext(cacheCtx, "failed to invalidate session cache after "+action,
			slog.String("user_id", userID.String()),
			slog.String("session_id", sessionID.String()),
			slog.Any("error", err),
		)
	}
}

func (s *Service) invalidateAllSessionsCache(ctx context.Context, userID uuid.UUID, sessionIDs []uuid.UUID, action string) {
	cacheCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()

	if len(sessionIDs) > 0 {
		if err := s.cache.DeleteBatch(cacheCtx, sessionIDs); err != nil {
			slog.WarnContext(cacheCtx, "failed to batch invalidate session cache after "+action,
				slog.String("user_id", userID.String()),
				slog.Int("session_count", len(sessionIDs)),
				slog.Any("error", err),
			)
		}
	}
}
