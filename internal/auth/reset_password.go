package auth

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/crypto"
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/user"
	"context"
	"net/http"
)

// --- RESET PASSWORD CONSTANTS ---

// Messages
const (
	MsgResetPasswordSuccess = "reset_password_success"
)

// Errors
const (
	ErrInvalidResetToken = "Invalid or expired reset token."
)

// --- RESET PASSWORD TYPES ---

type ResetPasswordRequest struct {
	Token       string `json:"token" validate:"required"`
	NewPassword string `json:"newPassword" validate:"required,min=8"`
}

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

	httpio.RespondOK(w, r, struct{}{}, MsgResetPasswordSuccess)

	return nil
}

// --- RESET PASSWORD SERVICE ---

// ResetPassword
func (s *Service) ResetPassword(ctx context.Context, tokenStr string, newPassword string) error {
	// Verify the token using the PasswordResetSecret
	claims, err := s.token.VerifyPasswordReset(tokenStr)
	if err != nil {
		return apperr.New(apperr.CodeUnauthenticated, ErrInvalidResetToken, apperr.WithErr(err))
	}

	// Hash the new password
	hashedPasswordBytes, err := crypto.HashPassword(newPassword)
	if err != nil {
		return apperr.New(apperr.CodeInternal, apperr.CodeInternal.Title(), apperr.WithErr(err))
	}

	// Execute update
	_, err = s.user.UpdatePassword(ctx, user.UpdatePasswordParams{
		ID:   claims.UserID,
		Hash: string(hashedPasswordBytes),
	})
	if err != nil {
		return err
	}

	return nil
}
