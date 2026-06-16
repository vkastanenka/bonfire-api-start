package auth

import (
	"bonfire-api/internal/httpio"
	"net/http"
)

// ForgotPassword initiates the password reset flow
func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) error {
	var data ForgotPasswordData

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
	httpio.RespondJSON(w, http.StatusOK, map[string]string{
		"message": "If an account exists with this email, a password reset link has been sent.",
	})

	return nil
}

// ResetPassword finalizes the password change
func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) error {
	var data ResetPasswordData

	if err := httpio.DecodeJSON(w, r, &data); err != nil {
		return err
	}

	if err := h.val.ValidateStruct(&data); err != nil {
		return err
	}

	if err := h.service.ResetPassword(r.Context(), data.Token, data.NewPassword); err != nil {
		return err
	}

	httpio.RespondJSON(w, http.StatusOK, map[string]string{
		"message": "Your password has been reset successfully. You may now log in.",
	})

	return nil
}
