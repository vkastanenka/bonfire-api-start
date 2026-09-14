package auth

import (
	"context"
	"errors"
	"time"

	"bonfire-api/internal/errs"
	"bonfire-api/internal/user"

	"github.com/google/uuid"
)

func (s *Service) VerifyEmail(ctx context.Context, userID uuid.UUID, token string) (*user.User, error) {
	claims, err := s.tokenProvider.VerifyEmailVerify(token)
	if err != nil {
		return nil, err
	}

	if userID != claims.UserID {
		return nil, errors.New("Token doesn't belong to you.")
	}

	if err := s.tokenCache.ConsumeEmailVerifyToken(ctx, claims); err != nil {
		return nil, err
	}

	now := time.Now()

	u, err := s.userRepo.Verify(ctx, userID, now, now)
	if err != nil {
		return nil, err
	}

	_ = s.userCache.Delete(ctx, userID)

	return u, nil
}

func (s *Service) ResendVerify(ctx context.Context, userID uuid.UUID) error {
	u, err := s.cachedUserRepo.Get(ctx, userID)
	if err != nil {
		if errs.IsNotFound(err) {
			return nil
		}
		return err
	}

	verifyToken, _, err := s.tokenProvider.GenerateEmailVerify(u.ID)
	if err != nil {
		return err
	}

	now := time.Now()
	payload := EventResendVerifyPayload{
		Email:    u.Email,
		Username: u.Username,
		Token:    verifyToken,
	}

	return s.outboxRepo.Publish(ctx, EventResendVerification, payload, now)
}
