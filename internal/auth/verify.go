package auth

import (
	"context"
	"time"

	"bonfire-api/internal/errs"
	"bonfire-api/internal/fields"

	"github.com/google/uuid"
)

// VerifyEmail verifies a user's email address using a signed verification token.
func (s *Service) VerifyEmail(ctx context.Context, tokenStr string) error {
	token, err := fields.ParseRequiredToken("token", tokenStr)
	if err != nil {
		return ErrVerificationTokenRequired()
	}

	claims, err := s.tokenProvider.VerifyEmailVerify(token.String())
	if err != nil {
		return err
	}

	u, err := s.userRepo.Get(ctx, claims.UserID)
	if err != nil {
		return err
	}

	now := fields.Now()

	return s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		var _, err = s.userRepo.Verify(txCtx, u.ID(), now, now)
		return err
	})
}

const (
	// Action label for rate limiting / cooldowns via ShieldStore
	ActionResendVerification = "retry-verification"

	// Cooldown window between verification resends
	ResendVerificationCooldown = 1 * time.Minute
)

type ResendVerificationEventPayload struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Token    string `json:"token"`
}

// ResendVerify sends a new account verification token if the user exists,
// and is not already verified.
func (s *Service) ResendVerify(ctx context.Context, rawUserID uuid.UUID) error {
	userID, err := fields.ParseRequiredID("id", rawUserID)
	if err != nil {
		return err
	}

	u, err := s.userRepo.Get(ctx, userID)
	if err != nil {
		if errs.IsNotFound(err) {
			return nil
		}
		return err
	}

	if u.IsVerified() {
		return nil
	}

	verifyToken, _, err := s.tokenProvider.GenerateEmailVerify(u.ID())
	if err != nil {
		return err
	}

	return s.outboxRepo.Publish(ctx, EventResendVerification, ResendVerificationEventPayload{
		Email:    u.Email().String(),
		Username: u.Username().String(),
		Token:    verifyToken,
	})
}
