package auth

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/sanitize"
	"bonfire-api/internal/worker"
	"context"
	"net/http"
)

// --- FORGOT PASSWORD TYPES ---

type ForgotPasswordReq struct {
	Email string `json:"email" validate:"auth_email"`
}

func (r *ForgotPasswordReq) Sanitize() {
	r.Email = sanitize.SanitizeEmail(r.Email)
}

// --- FORGOT PASSWORD CONSTANTS ---

const (
	msgForgotPasswordSuccess = "Forgot password flow started. Please check your email for next steps."
)

// --- FORGOT PASSWORD HANDLER ---

// ForgotPassword
func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) error {
	// Get JSON
	req, err := httpio.BindJSON[ForgotPasswordReq](w, r, h.validator)
	if err != nil {
		return err
	}

	// Call service
	if err := h.service.ForgotPassword(r.Context(), req.Email); err != nil {
		return err
	}

	// Respond
	httpio.RespondOK(w, r, struct{}{}, msgForgotPasswordSuccess)

	return nil
}

// --- FORGOT PASSWORD SERVICE ---

// ForgotPassword
func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	// Get user
	userAuth, err := s.user.GetAuthByEmail(ctx, email)
	if err != nil {
		// Respond ok if not found
		if apperr.IsNotFound(err) {
			return nil
		}
		return err
	}

	// Generate token
	resetToken, err := s.token.GeneratePasswordReset(userAuth.ID, userAuth.SecurityVersion)
	if err != nil {
		return apperr.NewInternal(err)
	}

	// Emit event
	if err := worker.EmitAuthForgotPassword(ctx, s.store, worker.ForgotPasswordPayload{
		Email: email,
		Token: resetToken,
	}); err != nil {
		return err
	}

	return nil
}
