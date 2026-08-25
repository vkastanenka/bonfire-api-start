package session

import (
	"context"
	"time"

	"bonfire-api/internal/fields"

	"github.com/google/uuid"
)

type Service struct {
	repo       Repository
	outboxRepo OutboxRepository
	tx         TX
}

func NewService(
	repo Repository,
	outboxRepo OutboxRepository,
	tx TX,
) *Service {
	return &Service{
		repo:       repo,
		outboxRepo: outboxRepo,
		tx:         tx,
	}
}

func (s *Service) ListValidByUserID(ctx context.Context, rawUserID uuid.UUID) ([]*Session, error) {
	userID, err := fields.ParseRequiredID("user_id", rawUserID)
	if err != nil {
		return nil, err
	}

	now := fields.Now()

	sessions, err := s.repo.ListValidByUserID(ctx, userID, now, listValidByUserIDLimit)
	if err != nil {
		return nil, err
	}

	sort(sessions)

	return sessions, nil
}

type RevokeParams struct {
	SessionID uuid.UUID
	UserID    uuid.UUID
}

func (s *Service) Revoke(ctx context.Context, rawID, rawUserID uuid.UUID) error {
	id, err := fields.ParseRequiredID("id", rawID)
	if err != nil {
		return err
	}

	userID, err := fields.ParseRequiredID("user_id", rawUserID)
	if err != nil {
		return err
	}

	now := fields.Now()

	return s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		if err := s.repo.Revoke(txCtx, id, userID, now); err != nil {
			return err
		}

		_, err := s.outboxRepo.Publish(txCtx, EventSessionRevoke, EventSessionRevokePayload{})
		if err != nil {
			return err
		}

		return nil
	})
}

func (s *Service) RevokeAll(ctx context.Context, rawUserID uuid.UUID) error {
	userID, err := fields.ParseRequiredID("user_id", rawUserID)
	if err != nil {
		return err
	}

	now := fields.Now()

	return s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		err := s.repo.RevokeAll(txCtx, userID, now)
		if err != nil {
			return err
		}

		_, err = s.outboxRepo.Publish(txCtx, EventSessionRevokeAll, EventSessionRevokeAllPayload{})
		if err != nil {
			return err
		}

		return nil
	})
}

func (s *Service) DeleteBatchExpired(ctx context.Context) error {
	now := time.Now()
	return s.repo.DeleteBatchExpired(ctx, now, deleteBatchExpiredLimit)
}
