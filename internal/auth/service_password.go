package auth

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/repository"
	"bonfire-api/internal/token"
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"
)

func (s *AuthService) ForgotPassword(ctx context.Context, email string) error {
	user, err := s.store.UserGetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return apperr.New(apperr.CodeInternal, "System error", apperr.WithErr(err))
	}

	// Generate a short-lived token (15 mins) specifically for resetting
	userID := uuid.UUID(user.ID.Bytes)
	resetToken, err := token.GenerateJWT(userID, s.tokenConfig.PasswordResetSecret, 15*time.Minute)
	if err != nil {
		return err
	}

	// Create Outbox Event
	jsonBytes, _ := json.Marshal(map[string]string{"email": email, "token": resetToken})
	_, err = s.store.OutboxEventCreate(ctx, repository.OutboxEventCreateParams{
		EventType: "user.forgot_password",
		Payload:   jsonBytes,
	})
	return err
}

func (s *AuthService) ResetPassword(ctx context.Context, tokenStr string, newPassword string) error {
	// Verify the token using the PasswordResetSecret
	claims, err := token.VerifyJWT(tokenStr, s.tokenConfig.PasswordResetSecret)
	if err != nil {
		return apperr.New(apperr.CodeUnauthenticated, "Invalid or expired reset token.")
	}

	// Hash the new password
	hashedPasswordBytes, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return apperr.New(apperr.CodeInternal, "Failed to hash password", apperr.WithErr(err))
	}

	// Execute update
	err = s.store.UserUpdatePassword(ctx, repository.UserUpdatePasswordParams{
		ID:           pgtype.UUID{Bytes: claims.UserID, Valid: true},
		PasswordHash: string(hashedPasswordBytes),
	})
	if err != nil {
		return apperr.New(apperr.CodeInternal, "Failed to update password", apperr.WithErr(err))
	}

	return nil
}
