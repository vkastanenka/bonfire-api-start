package session

import (
	"context"
	"log/slog"
	"time"

	"bonfire-api/internal/fields"
	"bonfire-api/internal/outbox"
	"bonfire-api/internal/redis"

	"github.com/google/uuid"
)

const (
	MaxUserSessions                = 10
	UserSessionsBatchLimit         = MaxUserSessions
	DefaultDeleteExpiredBatchLimit = 100
)

type Cache interface {
	Delete(ctx context.Context, sessionID fields.ID) error
	DeleteBatch(ctx context.Context, sessionIDs []fields.ID) error
}

type Repository interface {
	SessionUserGetBatch(ctx context.Context, userID fields.ID, now fields.Timestamp, limit int32) ([]*Session, error)
	SessionUserRevoke(ctx context.Context, id, userID fields.ID, revokedAt, updatedAt fields.Timestamp) error
	SessionUserRevokeAll(ctx context.Context, userID fields.ID, revokedAt, updatedAt fields.Timestamp) ([]fields.ID, error)
	SessionDeleteExpiredBatch(ctx context.Context, now time.Time, batchLimit int32) error
}

type OutboxRepository interface {
	Publish(ctx context.Context, variant string, payload any) (*outbox.Event, error)
}

type TX interface {
	ExecTx(ctx context.Context, fn func(txCtx context.Context) error) error
}

type Service struct {
	c  Cache
	r  Repository
	o  OutboxRepository
	tx TX
}

func NewService(
	c Cache,
	r Repository,
	o OutboxRepository,
	tx TX,
) *Service {
	return &Service{
		c:  c,
		r:  r,
		o:  o,
		tx: tx,
	}
}

func (s *Service) GetUserSessions(ctx context.Context, rawUserID uuid.UUID) ([]*Session, error) {
	userID, err := fields.ParseRequiredID("user_id", rawUserID)
	if err != nil {
		return nil, err
	}

	now := fields.NewTimestampFromTime(time.Now())
	sessions, err := s.r.SessionUserGetBatch(ctx, userID, now, UserSessionsBatchLimit)
	if err != nil {
		return nil, err
	}

	return sessions, nil
}

type RevokeParams struct {
	SessionID uuid.UUID
	UserID    uuid.UUID
}

func (s *Service) UserRevoke(ctx context.Context, p RevokeParams) error {
	id, err := fields.ParseRequiredID("id", p.SessionID)
	if err != nil {
		return err
	}

	userID, err := fields.ParseRequiredID("user_id", p.UserID)
	if err != nil {
		return err
	}

	now := fields.NewTimestampFromTime(time.Now())

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		if err := s.r.SessionUserRevoke(txCtx, id, userID, now, now); err != nil {
			return err
		}

		payload := EventSessionRevokePayload{
			SessionID: id.String(),
			UserID:    userID.String(),
			RevokedAt: now.String(),
		}

		if _, err := s.o.Publish(txCtx, EventSessionRevoke, payload); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	s.invalidateCache(ctx, id, "revoke session")

	return nil
}

func (s *Service) UserRevokeAll(ctx context.Context, rawUserID uuid.UUID) error {
	userID, err := fields.ParseRequiredID("user_id", rawUserID)
	if err != nil {
		return err
	}

	now := fields.NewTimestampFromTime(time.Now())

	var revokedSessionIDs []fields.ID

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		var err error
		revokedSessionIDs, err = s.r.SessionUserRevokeAll(txCtx, userID, now, now)
		if err != nil {
			return err
		}

		sessionIDsStr := make([]string, 0, len(revokedSessionIDs))
		for _, id := range revokedSessionIDs {
			sessionIDsStr = append(sessionIDsStr, id.String())
		}

		payload := EventSessionRevokeAllPayload{
			UserID:     userID.String(),
			SessionIDs: sessionIDsStr,
			RevokedAt:  now.String(),
		}

		if _, err := s.o.Publish(txCtx, EventSessionRevokeAll, payload); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	if s.c != nil && len(revokedSessionIDs) > 0 {
		if cacheErr := s.c.DeleteBatch(ctx, revokedSessionIDs); cacheErr != nil {
			slog.WarnContext(ctx, "failed to invalidate session cache batch on revoke all",
				"user_id", userID.String(),
				"count", len(revokedSessionIDs),
				"error", cacheErr,
				"scope", redis.ScopeSession,
			)
		}
	}

	return nil
}

func (s *Service) DeleteExpiredBatch(ctx context.Context) error {
	now := time.Now()
	return s.r.SessionDeleteExpiredBatch(ctx, now, DefaultDeleteExpiredBatchLimit)
}

func (s *Service) invalidateCache(ctx context.Context, id fields.ID, action string) {
	if s.c == nil {
		return
	}
	if err := s.c.Delete(ctx, id); err != nil {
		slog.WarnContext(ctx, "failed to invalidate session cache",
			"session_id", id.String(),
			"action", action,
			"error", err,
			"scope", redis.ScopeSession,
		)
	}
}
