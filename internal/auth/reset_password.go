package auth

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/cache"
	"bonfire-api/internal/crypto"
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/user"
	"context"
	"log/slog"
	"net/http"
)

// --- RESET PASSWORD TYPES ---

type ResetPasswordRequest struct {
	Token       string `json:"token" validate:"required"`
	NewPassword string `json:"new_password" validate:"security.password"`
}

// --- RESET PASSWORD CONSTANTS ---

// Messages
const (
	msgResetPasswordSuccess = "Password has been reset."
)

// Errors
const (
	errInvalidResetToken = "Invalid or expired reset token."
)

// --- RESET PASSWORD HANDLER ---

// ResetPassword
func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) error {
	// Get JSON
	req, err := httpio.BindJSON[ResetPasswordRequest](w, r, h.validator)
	if err != nil {
		return err
	}

	if err := h.service.ResetPassword(r.Context(), req.Token, req.NewPassword); err != nil {
		return err
	}

	httpio.RespondOK(w, r, struct{}{}, msgResetPasswordSuccess)

	return nil
}

// --- RESET PASSWORD SERVICE ---

// ResetPassword
func (s *Service) ResetPassword(ctx context.Context, tokenStr string, newPassword string) error {
	// Verify the token using the PasswordResetSecret
	claims, err := s.token.VerifyPasswordReset(tokenStr)
	if err != nil {
		return apperr.New(apperr.CodeUnauthorized, errInvalidResetToken, apperr.WithErr(err))
	}

	// Fetch user to check current version
	userAuth, err := s.user.GetAuthByID(ctx, claims.UserID)
	if err != nil {
		return err
	}

	// Allow only active users
	if !userAuth.IsActive() {
		return nil
	}

	// Validate security version
	if claims.SecurityVersion != userAuth.SecurityVersion {
		return apperr.New(apperr.CodeUnauthorized, errInvalidResetToken)
	}

	// Hash the new password
	hashedPasswordBytes, err := crypto.HashPassword(newPassword)
	if err != nil {
		return apperr.NewInternal(err)
	}

	// Persistence boundary
	persistCtx := context.WithoutCancel(ctx)

	// Execute update
	_, err = s.user.UpdatePassword(persistCtx, user.UpdatePasswordParams{
		ID:           claims.UserID,
		PasswordHash: string(hashedPasswordBytes),
	})
	if err != nil {
		return err
	}

	// Clear Brute-Force State (Lifts any active login bans/counters)
	failureKey := cache.LoginFailuresKey(userAuth.Email)
	lockoutKey := cache.LoginLockoutKey(userAuth.Email)

	if err := s.cache.Delete(persistCtx, failureKey); err != nil {
		slog.WarnContext(persistCtx, "failed to clear login failures on password reset", "error", err, "email", userAuth.Email)
	}
	if err := s.cache.Delete(persistCtx, lockoutKey); err != nil {
		slog.WarnContext(persistCtx, "failed to lift login lockout on password reset", "error", err, "email", userAuth.Email)
	}

	return nil
}
