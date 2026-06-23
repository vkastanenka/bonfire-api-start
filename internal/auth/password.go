package auth

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/user"
	"bonfire-api/internal/worker"
	"context"
	"net/http"

	"golang.org/x/crypto/bcrypt"
)

// --- Forgot Password DTO ---

type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// --- Forgot Password Handler ---

// ForgotPassword
func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) error {
	var data ForgotPasswordRequest

	if err := httpio.DecodeJSON(w, r, &data); err != nil {
		return err
	}

	if err := h.validator.ValidateStruct(&data); err != nil {
		return err
	}

	// Pass to service. We don't care if the user doesn't exist,
	// the service handles the logic silently to prevent enumeration.
	if err := h.service.ForgotPassword(r.Context(), data.Email); err != nil {
		return err
	}

	// Success response is generic to prevent email enumeration
	httpio.RespondOK(w, r, struct{}{}, "If an account exists with this email, a password reset link has been sent.")

	return nil
}

// --- Forgot Password Service ---

// ForgotPassword
func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	user, err := s.user.GetByEmail(ctx, email)
	if err != nil {
		return err
	}

	// Generate a short-lived token (15 mins) specifically for resetting
	resetToken, err := s.generatePasswordResetToken(user.ID)
	if err != nil {
		return err
	}

	// // Create Outbox Event
	err = worker.EmitEvent(ctx, s.store, worker.EventUserRegistered, worker.AuthForgotPasswordPayload{
		Email: email,
		Token: resetToken,
	})
	if err != nil {
		return err
	}

	return nil
}

// --- Reset Password DTO ---

type ResetPasswordRequest struct {
	Token       string `json:"token" validate:"required"`
	NewPassword string `json:"newPassword" validate:"required,min=8"`
}

// --- Reset Password Handler ---

// ResetPassword
func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) error {
	var data ResetPasswordRequest

	if err := httpio.DecodeJSON(w, r, &data); err != nil {
		return err
	}

	if err := h.validator.ValidateStruct(&data); err != nil {
		return err
	}

	row, err := h.service.ResetPassword(r.Context(), data.Token, data.NewPassword)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, row, "Your password has been reset successfully. You may now log in.")

	return nil
}

// --- Reset Password Service ---

// ResetPassword
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
