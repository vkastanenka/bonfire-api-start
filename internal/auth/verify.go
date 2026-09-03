package auth

import (
	"context"
	"errors"

	"bonfire-api/internal/errs"
	"bonfire-api/internal/fields"
	"bonfire-api/internal/user"

	"github.com/google/uuid"
)

func (s *Service) VerifyEmail(ctx context.Context, rawUserID uuid.UUID, tokenStr string) (*user.User, error) {
	userID, err := fields.ParseRequiredID("id", rawUserID)
	if err != nil {
		return nil, err
	}

	token, err := fields.ParseRequiredToken("token", tokenStr)
	if err != nil {
		return nil, ErrVerificationTokenRequired()
	}

	claims, err := s.tokenProvider.VerifyEmailVerify(token.String())
	if err != nil {
		return nil, err
	}

	if !userID.Equals(claims.UserID) {
		return nil, errors.New("Token doesn't belong to you.")
	}

	if err := s.tokenCache.ConsumeEmailVerifyToken(ctx, claims); err != nil {
		return nil, err
	}

	now := fields.Now()

	u, err := s.userRepo.Verify(ctx, userID, now, now)
	if err != nil {
		return nil, err
	}

	_ = s.userCache.Delete(ctx, userID)

	return u, nil
}

func (s *Service) ResendVerify(ctx context.Context, rawUserID uuid.UUID) error {
	userID, err := fields.ParseRequiredID("id", rawUserID)
	if err != nil {
		return err
	}

	u, err := s.userSvc.Get(ctx, userID.UUID())
	if err != nil {
		if errs.IsNotFound(err) {
			return nil
		}
		return err
	}

	verifyToken, _, err := s.tokenProvider.GenerateEmailVerify(u.ID())
	if err != nil {
		return err
	}

	now := fields.Now()
	payload := EventResendVerifyPayload{
		Email:    u.Email().String(),
		Username: u.Username().String(),
		Token:    verifyToken,
	}

	return s.outboxRepo.Publish(ctx, EventResendVerification, payload, now)
}
