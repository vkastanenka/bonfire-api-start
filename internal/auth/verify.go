package auth

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"bonfire-api/internal/errs"
)

// VerifyEmail verifies a user's email address using a signed verification token.
func (s *Service) VerifyEmail(ctx context.Context, tokenStr string) error {
	// 1. Guard Input
	if tokenStr == "" {
		return errs.InvalidArgument("Verification token is required.").
			FieldViolation("token", "Verification token is required.", "REQUIRED").
			Wrap(errors.New("verification token is required"))
	}

	// 2. Verify Email Token Signature & Claims
	claims, err := s.tokens.VerifyEmailVerify(tokenStr)
	if err != nil {
		return errs.Unauthenticated("Invalid or expired verification token.").
			FieldViolation("token", "Invalid or expired verification token.", "INVALID_TOKEN").
			Wrap(err)
	}

	// 3. Single-Use Token Check (Shield Store)
	// Prevents token replay attacks by checking if the token's JTI was already consumed
	consumed, err := s.shield.IsTokenConsumed(ctx, claims.ID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to check token consumed state",
			"error", err,
			"token_id", claims.ID,
			"user_id", claims.UserID,
		)
	} else if consumed {
		return errs.Unauthenticated("Verification token has already been used.").
			FieldViolation("token", "Verification token has already been used.", "TOKEN_ALREADY_USED").
			Wrap(errors.New("verification token already used"))
	}

	// 4. Fetch User Aggregate directly from UserRepository
	u, err := s.users.Get(ctx, claims.UserID)
	if err != nil {
		if errs.IsNotFound(err) {
			return errs.Unauthenticated("Invalid or expired verification token.").
				FieldViolation("token", "User associated with this token no longer exists.", "USER_NOT_FOUND").
				Wrap(err)
		}
		return err
	}

	// 5. Idempotency Check
	// If already verified, mark token as consumed and exit early successfully
	if u.IsVerified() {
		s.consumeTokenNonBlocking(ctx, claims.ID, claims.ExpiresAt.Time)
		return nil
	}

	// 6. Mutate User Domain Aggregate State
	u.Verify(time.Now().UTC())

	// 7. TRANSACTION: Atomically persist user verification state
	persistCtx := context.WithoutCancel(ctx)

	txErr := s.tx.ExecTx(persistCtx, func(txCtx context.Context) error {
		return s.users.Save(txCtx, u)
	})

	if txErr != nil {
		return txErr
	}

	// 8. Consume Token non-blockingly for remaining TTL
	s.consumeTokenNonBlocking(persistCtx, claims.ID, claims.ExpiresAt.Time)

	return nil
}

// consumeTokenNonBlocking marks a single-use token as consumed until its expiration.
func (s *Service) consumeTokenNonBlocking(ctx context.Context, tokenID string, expiresAt time.Time) {
	remainingTTL := time.Until(expiresAt)
	if remainingTTL <= 0 {
		return
	}

	if err := s.shield.MarkTokenConsumed(ctx, tokenID, remainingTTL); err != nil {
		slog.WarnContext(ctx, "failed to mark verification token as consumed",
			"error", err,
			"token_id", tokenID,
		)
	}
}
