package auth

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"bonfire-api/internal/errs"
	"bonfire-api/internal/outbox"

	"github.com/google/uuid"
)

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
// is not already verified, and is not on cooldown.
func (s *Service) ResendVerify(ctx context.Context, userID uuid.UUID) error {
	// 1. Guard Against Empty Input
	if userID == uuid.Nil {
		return errs.InvalidArgument("Invalid user ID.").
			FieldViolation("user_id", "User ID cannot be empty.", "REQUIRED").
			Wrap(errors.New("user ID cannot be empty"))
	}

	// 2. Cooldown Guard (Rate Limiting) via ShieldStore
	// Silent return on cooldown prevents info disclosure / enumeration
	onCooldown, err := s.shield.GetCooldown(ctx, "auth", ActionResendVerification, userID.String())
	if err != nil {
		slog.ErrorContext(ctx, "resend verification cooldown lookup failed",
			"error", err,
			"user_id", userID,
		)
	} else if onCooldown {
		return nil
	}

	// 3. Fetch User Aggregate directly from UserRepository
	// Silent return if user does not exist to prevent account enumeration
	u, err := s.users.Get(ctx, userID)
	if err != nil {
		if errs.IsNotFound(err) {
			return nil
		}
		return err
	}

	// 4. Idempotency Check: Don't resend if already verified
	if u.IsVerified() {
		return nil
	}

	// 5. Generate Verification Token
	verifyToken, _, err := s.tokens.GenerateEmailVerify(u.ID())
	if err != nil {
		return errs.Internal("failed to generate email verification token").Wrap(err)
	}

	// 6. Publish Event via Outbox Repository
	persistCtx := context.WithoutCancel(ctx)

	_, err = s.outbox.Publish(persistCtx, outbox.PublishParams{
		Variant: EventResendVerification,
		Payload: ResendVerificationEventPayload{
			Email:    u.Email().String(),
			Username: u.Username().String(),
			Token:    verifyToken,
		},
	})
	if err != nil {
		return err
	}

	// 7. Apply Cooldown Non-blocking
	if err := s.shield.SetCooldown(persistCtx, "user", ActionResendVerification, userID.String(), ResendVerificationCooldown); err != nil {
		slog.WarnContext(persistCtx, "failed to set resend verification cooldown",
			"error", err,
			"user_id", userID,
		)
	}

	return nil
}
