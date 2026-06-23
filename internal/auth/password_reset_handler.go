package auth

import (
	"bonfire-api/internal/httpio"
	"net/http"
)

// ResetPassword finalizes the password change
func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) error {
	var data ResetPasswordRequest

	if err := httpio.DecodeJSON(w, r, &data); err != nil {
		return err
	}

	if err := h.validator.ValidateStruct(&data); err != nil {
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
