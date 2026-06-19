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
func (s *AuthService) RotateTokens(ctx context.Context, oldTokenString string) (RefreshResponse, error) {
	// Check old token
	claims, err := s.tokenManager.VerifyJWT(oldTokenString, s.tokenConfig.RefreshSecret)
	if err != nil {
		return RefreshResponse{}, apperr.New(apperr.CodeUnauthenticated, "Invalid or expired session.")
	}

	// Check session
	sessionIDStr := claims.GetString("sid")
	if sessionIDStr == "" {
		return RefreshResponse{}, apperr.New(apperr.CodeUnauthenticated, "Malformed session payload.")
	}

	// Parse session id
	sessionUUID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		return RefreshResponse{}, apperr.New(apperr.CodeUnauthenticated, "Invalid session identifier.")
	}

	// Get user session
	session, err := s.store.UserSessionGetByID(ctx, pgtype.UUID{Bytes: sessionUUID, Valid: true})
	if err != nil {
		if repository.IsNotFoundError(err) {
			return RefreshResponse{}, apperr.New(apperr.CodeUnauthenticated, "Session no longer exists.")
		}
		return RefreshResponse{}, apperr.NewDBError(err)
	}

	// Check for an un-rotated token
	if session.RefreshToken != oldTokenString {
		_ = s.store.UserSessionMarkBlocked(ctx, session.ID)
		return RefreshResponse{}, apperr.New(apperr.CodeUnauthenticated, "Security breach detected. Session revoked.")
	}

	// Check if session blocked
	if session.IsBlocked {
		return RefreshResponse{}, apperr.New(apperr.CodeUnauthenticated, "Access denied. Session is blocked.")
	}

	// Check if session expired
	if time.Now().After(session.ExpiresAt.Time) {
		return RefreshResponse{}, apperr.New(apperr.CodeUnauthenticated, "Session expired. Please log in again.")
	}

	// Format userID
	userID := uuid.UUID(session.UserID.Bytes)

	// Get userRow
	userRow, err := s.store.UserGetByID(ctx, session.UserID)
	if err != nil {
		return RefreshResponse{}, apperr.NewDBError(err)
	}

	userIsVerified := userRow.VerifiedAt.Valid
	userRole := string(userRow.Role)

	// Generate new access token
	newAccessToken, err := s.generateAccessToken(userID, userRole, userIsVerified)
	if err != nil {
		return RefreshResponse{}, err
	}

	// Generate new refresh token
	newRefreshToken, err := s.generateRefreshToken(userID, sessionUUID)
	if err != nil {
		return RefreshResponse{}, err
	}

	// Update refresh token
	err = s.store.UserSessionUpdateRefreshToken(ctx, repository.UserSessionUpdateRefreshTokenParams{
		ID:           session.ID,
		RefreshToken: newRefreshToken,
		ExpiresAt: pgtype.Timestamptz{
			Time:  time.Now().Add(7 * 24 * time.Hour),
			Valid: true,
		},
	})
	if err != nil {
		return RefreshResponse{}, apperr.NewDBError(err)
	}

	// Return new tokens
	return RefreshResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
	}, nil
}
