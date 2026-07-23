// auth/refresh.go
package auth

import (
	"context"
	"crypto/subtle"
	"errors"
	"log/slog"
	"strings"
	"time"

	"bonfire-api/internal/crypto"
	"bonfire-api/internal/errs"
	"bonfire-api/internal/session"

	"github.com/google/uuid"
)

type RefreshParams struct {
	RefreshToken string
}

type RefreshResult struct {
	AccessToken           string
	RefreshToken          string
	RefreshTokenExpiresAt time.Time
}

func (s *Service) Refresh(ctx context.Context, p RefreshParams) (RefreshResult, error) {
	tokenStr := strings.TrimSpace(p.RefreshToken)
	if tokenStr == "" {
		return RefreshResult{}, errs.Unauthenticated("Missing refresh token.").
			FieldViolation("refresh_token", "Refresh token is required.", "REQUIRED").
			Wrap(errors.New("missing refresh token"))
	}

	// 1. Verify token signature & parse claims
	claims, err := s.tokens.VerifyRefresh(tokenStr)
	if err != nil {
		return RefreshResult{}, errs.Unauthenticated("Invalid or expired refresh token.").
			FieldViolation("refresh_token", "Invalid or expired refresh token.", "INVALID_TOKEN").
			Wrap(err)
	}

	// 2. Fetch session (Cache-first with DB fallback)
	sess, err := s.getSession(ctx, claims.SessionID)
	if err != nil {
		if errs.IsNotFound(err) {
			return RefreshResult{}, errs.Unauthenticated("Session not found.").
				FieldViolation("refresh_token", "Session associated with this token no longer exists.", "SESSION_NOT_FOUND").
				Wrap(err)
		}
		return RefreshResult{}, err
	}

	persistCtx := context.WithoutCancel(ctx)

	// 3. Security Check: Refresh Token Reuse Detection
	presentedHash := crypto.HashToken(tokenStr)
	currentHash := sess.RefreshTokenHash().Bytes()

	if subtle.ConstantTimeCompare([]byte(presentedHash), []byte(currentHash)) != 1 {
		slog.WarnContext(persistCtx, "refresh token reuse detected: token hash mismatch",
			"session_id", sess.ID(),
			"user_id", sess.UserID(),
		)

		// Revoke in DB and evict from cache to contain breach
		// if revokeErr := s.sessions.Revoke(persistCtx, sess.ID()); revokeErr != nil {
		// 	slog.ErrorContext(persistCtx, "failed to revoke compromised session in database",
		// 		"error", revokeErr,
		// 		"session_id", sess.ID(),
		// 	)
		// }

		if delErr := s.sessionStore.Delete(persistCtx, sess.ID()); delErr != nil {
			slog.ErrorContext(persistCtx, "failed to evict compromised session from cache",
				"error", delErr,
				"session_id", sess.ID(),
			)
		}

		return RefreshResult{}, errs.Unauthenticated("Invalid refresh token.").
			FieldViolation("refresh_token", "Token reuse detected.", "TOKEN_REUSE").
			Wrap(errors.New("refresh token reuse detected"))
	}

	// 4. Validate session state
	if sess.IsRevoked() {
		return RefreshResult{}, errs.Unauthenticated("Session has been revoked.").
			FieldViolation("refresh_token", "Session has been revoked.", "SESSION_REVOKED").
			Wrap(errors.New("session revoked"))
	}

	if sess.IsExpired() {
		return RefreshResult{}, errs.Unauthenticated("Session has expired.").
			FieldViolation("refresh_token", "Session has expired.", "SESSION_EXPIRED").
			Wrap(errors.New("session expired"))
	}

	// 5. Generate new token pair
	tokenPair, err := s.tokens.GeneratePair(sess.UserID(), sess.ID())
	if err != nil {
		return RefreshResult{}, errs.Internal("failed to generate token pair").Wrap(err)
	}

	newHash, err := session.NewRefreshTokenHash(crypto.HashToken(tokenPair.Refresh))
	if err != nil {
		return RefreshResult{}, errs.Internal("failed to hash refresh token").Wrap(err)
	}

	// 6. Mutate domain entity
	if err := sess.RotateToken(newHash, tokenPair.RefreshExpiresAt); err != nil {
		return RefreshResult{}, errs.Internal("failed to rotate session token").Wrap(err)
	}

	// 7. Persist changes to DB and update Cache
	if err := s.sessions.Save(persistCtx, sess); err != nil {
		return RefreshResult{}, err
	}

	if err := s.sessionStore.Set(persistCtx, sess); err != nil {
		slog.WarnContext(persistCtx, "failed to update session cache during token refresh",
			"error", err,
			"session_id", sess.ID(),
		)
	}

	return RefreshResult{
		AccessToken:           tokenPair.Access,
		RefreshToken:          tokenPair.Refresh,
		RefreshTokenExpiresAt: tokenPair.RefreshExpiresAt,
	}, nil
}

// getSession attempts to retrieve the session from cache, falling back to database on miss or error.
func (s *Service) getSession(ctx context.Context, id uuid.UUID) (*session.Session, error) {
	sess, err := s.sessionStore.Get(ctx, id)
	if err == nil && sess != nil {
		return sess, nil
	}

	if err != nil {
		slog.WarnContext(ctx, "session cache lookup degraded; falling back to DB",
			"error", err,
			"session_id", id,
		)
	}

	sess, err = s.sessions.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	// Repopulate cache asynchronously / non-blockingly
	persistCtx := context.WithoutCancel(ctx)
	if setErr := s.sessionStore.Set(persistCtx, sess); setErr != nil {
		slog.WarnContext(persistCtx, "failed to repopulate session cache after DB fallback",
			"error", setErr,
			"session_id", sess.ID(),
		)
	}

	return sess, nil
}
