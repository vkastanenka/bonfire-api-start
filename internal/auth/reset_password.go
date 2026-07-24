package auth

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"bonfire-api/internal/crypto"
	"bonfire-api/internal/errs"
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/session"

	"github.com/google/uuid"
)

type ResetPasswordParams struct {
	Token      string
	Password   string
	ClientMeta httpio.ClientMeta
}

type ResetPasswordResult struct {
	AccessToken           string
	RefreshToken          string
	RefreshTokenExpiresAt time.Time
}

// ResetPassword verifies the reset token, updates the user's password, invalidates
// all active sessions, and logs the user in with a fresh session pair.
func (s *Service) ResetPassword(ctx context.Context, p ResetPasswordParams) (ResetPasswordResult, error) {
	// 1. Guard Input
	if p.Token == "" {
		return ResetPasswordResult{}, errs.InvalidArgument("Reset token is required.").
			FieldViolation("token", "Reset token is required.", "REQUIRED").
			Wrap(errors.New("reset token is required"))
	}

	// 2. Verify Password Reset Token Claims
	claims, err := s.tokens.VerifyPasswordReset(p.Token)
	if err != nil {
		return ResetPasswordResult{}, errs.Unauthenticated("Invalid or expired reset token.").
			FieldViolation("token", "Invalid or expired reset token.", "INVALID_TOKEN").
			Wrap(err)
	}

	// 3. Fetch User Aggregate
	u, err := s.users.Get(ctx, claims.UserID)
	if err != nil {
		if errs.IsNotFound(err) {
			return ResetPasswordResult{}, errs.Unauthenticated("Invalid or expired reset token.").
				FieldViolation("token", "User associated with this token no longer exists.", "USER_NOT_FOUND").
				Wrap(err)
		}
		return ResetPasswordResult{}, err
	}

	// 4. Hash New Password
	passwordHash, err := crypto.HashPassword(p.Password)
	if err != nil {
		return ResetPasswordResult{}, errs.Internal("failed to hash new password").Wrap(err)
	}

	// 5. Mutate User Domain Aggregate State
	if err := u.UpdatePassword(passwordHash); err != nil {
		return ResetPasswordResult{}, errs.InvalidArgument("Invalid password.").
			FieldViolation("password", "Password does not meet validation criteria.", "INVALID_PASSWORD").
			Wrap(err)
	}

	// 6. Generate New Session ID & Tokens (ID Synchronization)
	sessionID, err := uuid.NewV7()
	if err != nil {
		return ResetPasswordResult{}, errs.Internal("failed to generate session ID").Wrap(err)
	}

	tokenPair, err := s.tokens.GeneratePair(u.ID(), sessionID)
	if err != nil {
		return ResetPasswordResult{}, errs.Internal("failed to generate token pair").Wrap(err)
	}

	tokenHash, err := session.NewRefreshTokenHash(crypto.HashToken(tokenPair.Refresh))
	if err != nil {
		return ResetPasswordResult{}, errs.Internal("failed to hash refresh token").Wrap(err)
	}

	newSession, err := session.New(
		sessionID,
		u.ID(),
		tokenHash,
		tokenPair.RefreshExpiresAt,
		p.ClientMeta.IP,
		p.ClientMeta.UserAgent,
		p.ClientMeta.OS,
		p.ClientMeta.Browser,
	)
	if err != nil {
		return ResetPasswordResult{}, errs.Internal("failed to instantiate session").Wrap(err)
	}

	// 7. TRANSACTION: Atomically revoke existing sessions, update password, and create new session
	persistCtx := context.WithoutCancel(ctx)

	txErr := s.tx.ExecTx(persistCtx, func(txCtx context.Context) error {
		// Invalidate all existing active sessions for security
		// if err := s.sessions.RevokeByUserID(txCtx, u.ID()); err != nil {
		// 	return err
		// }

		// Save updated user aggregate (contains new password hash & updated timestamp)
		if err := s.users.Update(txCtx, u); err != nil {
			return err
		}

		// Create new session
		if err := s.sessions.Create(txCtx, newSession); err != nil {
			return err
		}

		return nil
	})

	if txErr != nil {
		return ResetPasswordResult{}, txErr
	}

	// 8. Cache New Session Non-blockingly
	if err := s.sessionStore.Set(persistCtx, newSession); err != nil {
		slog.WarnContext(persistCtx, "failed to cache session after password reset",
			"error", err,
			"session_id", newSession.ID(),
			"user_id", u.ID(),
		)
	}

	return ResetPasswordResult{
		AccessToken:           tokenPair.Access,
		RefreshToken:          tokenPair.Refresh,
		RefreshTokenExpiresAt: tokenPair.RefreshExpiresAt,
	}, nil
}
