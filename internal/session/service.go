package session

import (
	"context"

	"bonfire-api/internal/fields"

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

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		if err := s.repo.Revoke(txCtx, id, userID, now); err != nil {
			return err
		}

		payload := EventRevokePayload{
			SessionID: id.String(),
			UserID:    userID.String(),
			RevokedAt: now.String(),
		}

		return s.outboxRepo.Publish(txCtx, EventRevoke, payload, now)
	})
	if err != nil {
		return err
	}

	_ = s.cache.Delete(ctx, id)

	return nil
}

func (s *Service) RevokeAll(ctx context.Context, rawUserID uuid.UUID) error {
	userID, err := fields.ParseRequiredID("user_id", rawUserID)
	if err != nil {
		return err
	}

	now := fields.Now()

	activeSessions, err := s.repo.ListValidByUserID(ctx, userID, now, listValidByUserIDLimit)
	if err != nil {
		return err
	}

	sessionIDs := make([]fields.ID, len(activeSessions))
	sessionIDStrings := make([]string, len(activeSessions))
	for i, sess := range activeSessions {
		sessionIDs[i] = sess.ID()
		sessionIDStrings[i] = sess.ID().String()
	}

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		if _, err := s.repo.RevokeAll(txCtx, userID, now); err != nil {
			return err
		}

		payload := EventRevokeAllPayload{
			UserID:     userID.String(),
			SessionIDs: sessionIDStrings,
			RevokedAt:  now.String(),
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
