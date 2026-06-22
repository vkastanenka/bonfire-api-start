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
	userAuth, err := s.store.UserGetByEmail(ctx, r.Email)
	if err != nil {
		return LoginResult{}, apperr.NewDBError(err)
	}

	// Check password
	err = crypto.VerifyPassword(userAuth.PasswordHash, r.Password)
	if err != nil {
		return LoginResult{}, NewLoginCredentialsError()
	}

	// Issue new token bundle
	userID := uuid.UUID(userAuth.ID.Bytes)
	userRole := string(userAuth.Role)
	userIsVerified := userAuth.VerifiedAt.Valid

	bundle, err := s.tokenManager.IssueNewBundle(userID, userRole, userIsVerified)
	if err != nil {
		return LoginResult{}, err
	}

	// Create user session
	_, err = s.store.UserSessionCreate(ctx, repository.UserSessionCreateParams{
		ID:           pgtype.UUID{Bytes: bundle.SessionID, Valid: true},
		UserID:       userAuth.ID,
		RefreshToken: bundle.RefreshToken,
		UserAgent:    r.Meta.UserAgent,
		ClientIp:     r.Meta.IP,
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
		AccessToken:  bundle.AccessToken,
		RefreshToken: bundle.RefreshToken,
	}, nil
}
