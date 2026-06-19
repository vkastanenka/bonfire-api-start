package auth

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/crypto"
	"bonfire-api/internal/repository"
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// Login
func (s *AuthService) Login(ctx context.Context, r LoginParams) (LoginResult, error) {
	// Fetch user credentials
	userAuth, err := s.store.UserGetAuthCredentials(ctx, r.Email)
	if err != nil {
		// User not found
		if repository.IsNotFoundError(err) {
			return LoginResult{}, NewInvalidCredentialsErr()
		}

		return LoginResult{}, apperr.NewDBError(err)
	}

	// Check password
	err = crypto.VerifyPassword(userAuth.PasswordHash, r.Password)
	if err != nil {
		return LoginResult{}, NewInvalidCredentialsErr()
	}

	// Convert pgtype.UUID to uuid.UUID
	userID := uuid.UUID(userAuth.ID.Bytes)
	userIsVerified := userAuth.VerifiedAt.Valid
	userRole := string(userAuth.Role)

	// Generate Access Token (15 minutes)
	accessToken, err := s.generateAccessToken(userID, userRole, userIsVerified)
	if err != nil {
		return LoginResult{}, err
	}

	// Generate session ID
	sessionID, err := uuid.NewV7()
	if err != nil {
		return LoginResult{}, err
	}

	// Generate Refresh Token (7 days)
	refreshToken, err := s.generateRefreshToken(userID, sessionID)
	if err != nil {
		return LoginResult{}, err
	}

	// Create user session
	_, err = s.store.UserSessionCreate(ctx, repository.UserSessionCreateParams{
		ID:           pgtype.UUID{Bytes: sessionID, Valid: true},
		UserID:       userAuth.ID,
		RefreshToken: refreshToken,
		UserAgent:    r.UserAgent,
		ClientIp:     r.ClientIP,
		IsBlocked:    false,
		ExpiresAt: pgtype.Timestamptz{
			Time:  time.Now().Add(7 * 24 * time.Hour),
			Valid: true,
		},
	})
	if err != nil {
		return LoginResult{}, apperr.NewDBError(err)
	}

	// Return the tokens
	return LoginResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}
