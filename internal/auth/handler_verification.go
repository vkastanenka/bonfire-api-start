package auth

import (
	"bonfire-api/internal/httpio"
	"net/http"
)

type VerifyEmailRequest struct {
	Token string `json:"token" validate:"required"`
}

// VerifyEmail handles incoming verification tokens sent from the frontend client.
func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) error {
	var req VerifyEmailRequest

	if err := httpio.DecodeJSON(w, r, &req); err != nil {
		return err
	}

	if err := h.val.ValidateStruct(&req); err != nil {
		return err
	}

	// Pass the token to the service method you just wrote
	if err := h.service.VerifyEmail(r.Context(), req.Token); err != nil {
		return err
	}

	httpio.RespondJSON(w, http.StatusOK, map[string]string{
		"message": "Your email address has been successfully verified!",
	})

	return nil
}

type ResendVerificationEmailRequest struct {
	Email string `json:"email" validate:"required,email"`
}

func (h *AuthHandler) ResendVerificationEmail(w http.ResponseWriter, r *http.Request) error {
	var data ResendVerificationEmailRequest

	if err := httpio.DecodeJSON(w, r, &data); err != nil {
		return err
	}

	if err := h.val.ValidateStruct(&data); err != nil {
		return err
	}

	// Dispatch to service
	if err := h.service.ResendVerificationEmail(r.Context(), data.Email); err != nil {
		return err
	}

	// Return a generic 200 OK regardless of whether the email was found or not
	httpio.RespondJSON(w, http.StatusOK, map[string]string{
		"message": "If an unverified account exists with that email, a verification link has been sent.",
	})

	return nil
}
