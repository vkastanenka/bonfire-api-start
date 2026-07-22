// auth/refresh.go
package auth

import (
	"context"
	"crypto/subtle"
	"errors"
	"log/slog"
	"strings"
	"time"

	"bonfire-api/internal/apperr"
	"bonfire-api/internal/crypto"
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
		return RefreshResult{}, apperr.NewInvalidArgument(
			errors.New("missing refresh token"),
		)
	}

	// 1. Verify token signature & parse claims
	claims, err := s.tokens.VerifyRefresh(tokenStr)
	if err != nil {
		return RefreshResult{}, apperr.NewInvalidArgument(
			err,
		)
	}

	// 2. Fetch session (Cache-first with DB fallback)
	sess, err := s.getSession(ctx, claims.SessionID)
	if err != nil {
		if apperr.IsNotFound(err) {
			return RefreshResult{}, apperr.NewInvalidArgument(
				err,
			)
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
		if revokeErr := s.sessions.Revoke(persistCtx, sess.ID()); revokeErr != nil {
			slog.ErrorContext(persistCtx, "failed to revoke compromised session in database",
				"error", revokeErr,
				"session_id", sess.ID(),
			)
		}

		if delErr := s.sessionCache.Delete(persistCtx, sess.ID()); delErr != nil {
			slog.ErrorContext(persistCtx, "failed to evict compromised session from cache",
				"error", delErr,
				"session_id", sess.ID(),
			)
		}

		return RefreshResult{}, apperr.NewInvalidArgument(
			errors.New("refresh token reuse detected"),
		)
	}

	// 4. Validate session state
	if sess.IsRevoked() {
		return RefreshResult{}, apperr.NewInvalidArgument(
			errors.New("session revoked"),
		)
	}

	if sess.IsExpired() {
		return RefreshResult{}, apperr.NewInvalidArgument(
			errors.New("session expired"),
		)
	}

	// 5. Generate new token pair
	tokenPair, err := s.tokens.GeneratePair(sess.UserID(), sess.ID())
	if err != nil {
		return RefreshResult{}, apperr.NewInternal(err)
	}

	newHash, err := session.NewRefreshTokenHash(crypto.HashToken(tokenPair.Refresh))
	if err != nil {
		return RefreshResult{}, apperr.NewInternal(err)
	}

	// 6. Mutate domain entity
	if err := sess.RotateToken(newHash, tokenPair.RefreshExpiresAt); err != nil {
		return RefreshResult{}, apperr.NewPermissionDenied(err)
	}

	// 7. Persist changes to DB and update Cache
	if err := s.sessions.Update(persistCtx, sess); err != nil {
		return RefreshResult{}, err
	}

	if err := s.sessionCache.Set(persistCtx, sess); err != nil {
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
	sess, err := s.sessionCache.Get(ctx, id)
	if err == nil && sess != nil {
		return sess, nil
	}

	if err != nil {
		slog.WarnContext(ctx, "session cache lookup degraded; falling back to DB",
			"error", err,
			"session_id", id,
		)
	}

	sess, err = s.sessions.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Repopulate cache asynchronously / non-blockingly
	persistCtx := context.WithoutCancel(ctx)
	if setErr := s.sessionCache.Set(persistCtx, sess); setErr != nil {
		slog.WarnContext(persistCtx, "failed to repopulate session cache after DB fallback",
			"error", setErr,
			"session_id", sess.ID(),
		)
	}

	return sess, nil
}
