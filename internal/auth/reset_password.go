package auth

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/crypto"
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/user"
	"context"
	"net/http"
	"strings"
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
	NewPassword string `json:"new_password" validate:"required,min=8"`
}

// --- RESET PASSWORD HANDLER ---

// ResetPassword
func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) error {
	// 1. Get token from Header (Bearer token pattern)
	authHeader := r.Header.Get("Authorization")
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == "" {
		return apperr.New(apperr.CodeUnauthorized, "Missing authorization token")
	}

	// Get JSON
	req, err := httpio.BindJSON[ResetPasswordRequest](w, r, h.validator)
	if err != nil {
		return err
	}

	if err := h.service.ResetPassword(r.Context(), token, req.NewPassword); err != nil {
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
		return apperr.New(apperr.CodeUnauthorized, ErrInvalidResetToken, apperr.WithErr(err))
	}

	// Fetch user to check current version
	userAuth, err := s.user.GetAuthByID(ctx, claims.UserID)
	if err != nil {
		return err
	}

	// Validate security version
	if claims.SecurityVersion != userAuth.SecurityVersion {
		return apperr.New(apperr.CodeUnauthorized, ErrInvalidResetToken)
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
