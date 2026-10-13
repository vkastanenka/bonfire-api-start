package auth

import (
	"context"
	"time"

	"bonfire-api/internal/appctx"
	"bonfire-api/internal/outbox"

	"github.com/google/uuid"
)

type RevokeParams struct {
	UserID          uuid.UUID
	SessionID       uuid.UUID
	RevokeSessionID uuid.UUID
}

func (s *Service) RevokeSessionID(ctx context.Context, p RevokeParams) error {
	now := time.Now()

	err := s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		if err := s.sessionRepo.Revoke(txCtx, p.RevokeSessionID, p.UserID, now); err != nil {
			return err
		}

		event, err := outbox.New(
			EventSessionRevoked,
			outbox.NewMetadata(appctx.GetTraceID(ctx), &p.UserID, &p.SessionID, nil),
			newEventSessionRevokedPayload(p.UserID, p.SessionID, now),
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

	s.deleteSessionCache(ctx, p.UserID, p.SessionID, "revoke session")

	return nil
}

type RevokeAllParams struct {
	UserID    uuid.UUID
	SessionID uuid.UUID
}

func (s *Service) RevokeAll(ctx context.Context, p RevokeAllParams) error {
	var revokedSessionIDs []uuid.UUID

	now := time.Now()

	err := s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		var txErr error

		if revokedSessionIDs, txErr = s.sessionRepo.RevokeAll(txCtx, p.UserID, now); txErr != nil {
			return txErr
		}

		event, err := outbox.New(
			EventSessionRevokedAll,
			outbox.NewMetadata(appctx.GetTraceID(ctx), &p.UserID, &p.SessionID, nil),
			newEventSessionRevokedPayload(p.UserID, p.SessionID, now),
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

	s.deleteSessionBatchCache(ctx, p.UserID, revokedSessionIDs, "revoke all sessions")

	return nil
}
