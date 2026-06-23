package auth

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/user"
	"context"

	"golang.org/x/crypto/bcrypt"
)

func (s *Service) ResetPassword(ctx context.Context, tokenStr string, newPassword string) (user.View, error) {
	// Verify the token using the PasswordResetSecret
	claims, err := s.tokenManager.VerifyJWT(tokenStr, s.tokenConfig.PasswordResetSecret)
	if err != nil {
		return user.View{}, apperr.New(apperr.CodeUnauthenticated, "Invalid or expired reset token.")
	}

	// Hash the new password
	hashedPasswordBytes, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return user.View{}, apperr.New(apperr.CodeInternal, "Failed to hash password", apperr.WithErr(err))
	}

	// Execute update
	row, err := s.user.UpdatePassword(ctx, user.UpdatePasswordParams{
		ID:   claims.UserID,
		Hash: string(hashedPasswordBytes),
	})
	if err != nil {
		return user.View{}, apperr.New(apperr.CodeInternal, "Failed to update password", apperr.WithErr(err))
	}

	return row, nil
}
