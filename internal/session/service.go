package session

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Service struct {
	cache      Cache
	repo       Repository
	outboxRepo OutboxRepository
	tx         TX
}

func NewService(
	cache Cache,
	repo Repository,
	outboxRepo OutboxRepository,
	tx TX,
) *Service {
	return &Service{
		cache:      cache,
		repo:       repo,
		outboxRepo: outboxRepo,
		tx:         tx,
	}
}

func (s *Service) ListValidByUserID(ctx context.Context, userID uuid.UUID) ([]*Session, error) {
	now := time.Now()

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

func (s *Service) Revoke(ctx context.Context, p RevokeParams) error {
	now := time.Now()

	err := s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		if err := s.repo.Revoke(txCtx, p.SessionID, p.UserID, now); err != nil {
			return err
		}

		payload := EventRevokePayload{
			SessionID: p.SessionID,
			UserID:    p.UserID,
			RevokedAt: now,
		}

		return s.outboxRepo.Publish(txCtx, EventRevoke, payload, now)
	})
	if err != nil {
		return err
	}

	_ = s.cache.Delete(ctx, p.SessionID)

	return nil
}

func (s *Service) RevokeAll(ctx context.Context, userID uuid.UUID) error {
	now := time.Now()

	activeSessions, err := s.repo.ListValidByUserID(ctx, userID, now, listValidByUserIDLimit)
	if err != nil {
		return err
	}

	sessionIDs := make([]uuid.UUID, len(activeSessions))
	for i, sess := range activeSessions {
		sessionIDs[i] = sess.ID
	}

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		if _, err := s.repo.RevokeAll(txCtx, userID, now); err != nil {
			return err
		}

		payload := EventRevokeAllPayload{
			UserID:     userID,
			SessionIDs: sessionIDs,
			RevokedAt:  now,
		}

		return s.outboxRepo.Publish(txCtx, EventRevokeAll, payload, now)
	})
	if err != nil {
		return err
	}

	if len(sessionIDs) > 0 {
		_ = s.cache.DeleteBatch(ctx, sessionIDs)
	}

	return nil
}
