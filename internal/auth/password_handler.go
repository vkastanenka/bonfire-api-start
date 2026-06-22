package auth

import (
	"bonfire-api/internal/httpio"
	"net/http"
)

type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// ForgotPassword initiates the password reset flow
func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) error {
	var data ForgotPasswordRequest

	if err := httpio.DecodeJSON(w, r, &data); err != nil {
		return err
	}

	if err := h.val.ValidateStruct(&data); err != nil {
		return err
	}

	// Pass to service. We don't care if the user doesn't exist,
	// the service handles the logic silently to prevent enumeration.
	if err := h.service.ForgotPassword(r.Context(), data.Email); err != nil {
		return err
	}

	// Success response is generic to prevent email enumeration
	httpio.RespondJSON(w, r, http.StatusOK, map[string]string{
		"message": "If an account exists with this email, a password reset link has been sent.",
	})

	return nil
}

type ResetPasswordRequest struct {
	Token       string `json:"token" validate:"required"`
	NewPassword string `json:"newPassword" validate:"required,min=8"`
}

// ResetPassword finalizes the password change
func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) error {
	var data ResetPasswordRequest

	if err := httpio.DecodeJSON(w, r, &data); err != nil {
		return err
	}

	if err := h.val.ValidateStruct(&data); err != nil {
		return err
	}

	if err := h.service.ResetPassword(r.Context(), data.Token, data.NewPassword); err != nil {
		return err
	}

	httpio.RespondJSON(w, r, http.StatusOK, map[string]string{
		"message": "Your password has been reset successfully. You may now log in.",
	})

	return nil
}
