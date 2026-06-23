package auth

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/crypto"
	"context"
	"time"

	"github.com/google/uuid"
)

// Login
func (s *Service) Login(ctx context.Context, r LoginParams) (LoginResult, error) {
	// Fetch user credentials
	userAuth, err := s.user.GetAuthByEmail(ctx, r.Email)
	if err != nil {
		return LoginResult{}, apperr.NewDBError(err)
	}

	// Check password
	err = crypto.VerifyPassword(userAuth.PasswordHash, r.Password)
	if err != nil {
		return LoginResult{}, NewLoginCredentialsError()
	}

	// Issue new token bundle
	userID := uuid.UUID(userAuth.ID)
	userRole := string(userAuth.Role)
	userIsVerified := userAuth.VerifiedAt != nil

	bundle, err := s.tokenManager.IssueNewBundle(userID, userRole, userIsVerified)
	if err != nil {
		return LoginResult{}, err
	}

	// Create user session
	_, err = s.CreateUserSession(ctx, CreateUserSessionParams{
		ID:           bundle.SessionID,
		UserID:       userID,
		RefreshToken: bundle.RefreshToken,
		UserAgent:    r.Meta.UserAgent,
		ClientIP:     r.Meta.IP,
		IsBlocked:    false,
		ExpiresAt:    time.Now().Add(7 * 24 * time.Hour),
	})
	if err != nil {
		return LoginResult{}, err
	}

	// Return the tokens
	return LoginResult{
		AccessToken:  bundle.AccessToken,
		RefreshToken: bundle.RefreshToken,
	}, nil
}
