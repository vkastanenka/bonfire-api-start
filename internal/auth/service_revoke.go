package auth

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type RevokeSessionParams struct {
	UserID          uuid.UUID
	RevokeSessionID uuid.UUID
}

func (s *Service) RevokeSession(ctx context.Context, p RevokeSessionParams) error {
	now := time.Now()

	if err := s.sessionRepo.Revoke(ctx, p.RevokeSessionID, p.UserID, now); err != nil {
		return err
	}

	s.cacheDeleteSession(ctx, p.UserID, p.RevokeSessionID, "revoke session")
	s.broadcastSessionRevoked(ctx, p.UserID, p.RevokeSessionID, p)

	return nil
}

type RevokeAllSessionsParams struct {
	UserID uuid.UUID
}

func (s *Service) RevokeAllSessions(ctx context.Context, p RevokeAllSessionsParams) error {
	now := time.Now()

	revokedSessionIDs, err := s.sessionRepo.RevokeAll(ctx, p.UserID, now)
	if err != nil {
		return err
	}

	s.cacheDeleteSessionBatch(ctx, p.UserID, revokedSessionIDs, "revoke all sessions")
	s.broadcastAllSessionsRevoked(ctx, p.UserID)

	return nil
}
