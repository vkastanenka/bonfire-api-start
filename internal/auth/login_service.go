package auth

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/crypto"
	"bonfire-api/internal/repository"
	"context"
	"net/netip"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// Login
func (s *AuthService) Login(ctx context.Context, req LoginInput, userAgent string, clientIP netip.Addr) (LoginResponse, error) {
	// Set up invalid params for error handling
	invalidParams := apperr.WithInvalidParams([]apperr.InvalidParam{
		{Name: "email", Reason: "Invalid credentials."},
		{Name: "password", Reason: "Invalid credentials."},
	})

	// Fetch user credentials
	userAuth, err := s.store.UserGetAuthCredentials(ctx, req.Email)
	if err != nil {
		// User not found
		if repository.IsNotFoundError(err) {
			return LoginResponse{}, apperr.New(apperr.CodeNotFound, "Invalid credentials.", invalidParams)
		}

		return LoginResponse{}, apperr.NewDBError(err)
	}

	// Check password
	err = crypto.VerifyPassword(userAuth.PasswordHash, req.Password)
	if err != nil {
		return LoginResponse{}, apperr.New(apperr.CodeUnauthenticated, "Invalid credentials.", invalidParams)
	}

	// Convert pgtype.UUID to uuid.UUID
	userID := uuid.UUID(userAuth.ID.Bytes)
	userIsVerified := userAuth.VerifiedAt.Valid
	userRole := string(userAuth.Role)

	// Generate Access Token (15 minutes)
	accessToken, err := s.generateAccessToken(userID, userRole, userIsVerified)
	if err != nil {
		return LoginResponse{}, err
	}

	// Generate session ID
	sessionID, err := uuid.NewV7()
	if err != nil {
		return LoginResponse{}, err
	}

	// Generate Refresh Token (7 days)
	refreshToken, err := s.generateRefreshToken(userID, sessionID)
	if err != nil {
		return LoginResponse{}, err
	}

	// Create user session
	_, err = s.store.UserSessionCreate(ctx, repository.UserSessionCreateParams{
		ID:           pgtype.UUID{Bytes: sessionID, Valid: true},
		UserID:       userAuth.ID,
		RefreshToken: refreshToken,
		UserAgent:    userAgent,
		ClientIp:     clientIP,
		IsBlocked:    false,
		ExpiresAt: pgtype.Timestamptz{
			Time:  time.Now().Add(7 * 24 * time.Hour),
			Valid: true,
		},
	})
	if err != nil {
		return LoginResponse{}, apperr.NewDBError(err)
	}

	// Return the tokens
	return LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}
