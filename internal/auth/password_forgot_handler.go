package auth

import (
	"bonfire-api/internal/httpio"
	"net/http"
)

// ForgotPassword initiates the password reset flow
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
