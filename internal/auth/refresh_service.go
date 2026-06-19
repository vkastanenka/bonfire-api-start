package auth

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/repository"
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// RotateTokens
func (s *AuthService) RotateTokens(ctx context.Context, r RefreshParams) (RefreshResult, error) {
	// Check old token
	claims, err := s.tokenManager.VerifyJWT(r.RefreshToken, s.tokenConfig.RefreshSecret)
	if err != nil {
		return RefreshResult{}, apperr.New(apperr.CodeUnauthenticated, ErrSessionInvalid)
	}

	// Check session
	sessionIDStr := claims.GetString("sid")
	if sessionIDStr == "" {
		return RefreshResult{}, apperr.New(apperr.CodeUnauthenticated, ErrSessionMalformed)
	}

	// Parse session id
	sessionUUID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		return RefreshResult{}, apperr.New(apperr.CodeUnauthenticated, ErrSessionInvalid)
	}

	// Get user session
	session, err := s.store.UserSessionGetByID(ctx, pgtype.UUID{Bytes: sessionUUID, Valid: true})
	if err != nil {
		if repository.IsNotFoundError(err) {
			return RefreshResult{}, apperr.New(apperr.CodeUnauthenticated, ErrSessionNotFound)
		}
		return RefreshResult{}, apperr.NewDBError(err)
	}

	// Check for an un-rotated token
	if session.RefreshToken != r.RefreshToken {
		_ = s.store.UserSessionMarkBlocked(ctx, session.ID)
		return RefreshResult{}, apperr.New(apperr.CodeUnauthenticated, ErrSessionInvalid)
	}

	// Check if session blocked
	if session.IsBlocked {
		return RefreshResult{}, apperr.New(apperr.CodeUnauthenticated, ErrSessionBlocked)
	}

	// Check if session expired
	if time.Now().After(session.ExpiresAt.Time) {
		return RefreshResult{}, apperr.New(apperr.CodeUnauthenticated, ErrSessionExpired)
	}

	// Format userID
	userID := uuid.UUID(session.UserID.Bytes)

	// Get userRow
	userRow, err := s.store.UserGetByID(ctx, session.UserID)
	if err != nil {
		return RefreshResult{}, apperr.NewDBError(err)
	}

	// Issue new token bundle
	userRole := string(userRow.Role)
	userIsVerified := userRow.VerifiedAt.Valid

	bundle, err := s.tokenManager.IssueNewBundle(userID, userRole, userIsVerified)
	if err != nil {
		return RefreshResult{}, err
	}

	// Update refresh token
	err = s.store.UserSessionUpdateRefreshToken(ctx, repository.UserSessionUpdateRefreshTokenParams{
		ID:           session.ID,
		RefreshToken: bundle.RefreshToken,
		ExpiresAt: pgtype.Timestamptz{
			Time:  time.Now().Add(7 * 24 * time.Hour),
			Valid: true,
		},
	})
	if err != nil {
		return RefreshResult{}, apperr.NewDBError(err)
	}

	// Return new tokens
	return RefreshResult{
		AccessToken:  bundle.AccessToken,
		RefreshToken: bundle.RefreshToken,
	}, nil
}
