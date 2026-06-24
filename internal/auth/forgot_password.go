package auth

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/worker"
	"context"
	"net/http"
)

// --- FORGOT PASSWORD CONSTANTS ---

const (
	MsgForgotPasswordSuccess = "forgot_password_success"
)

// --- FORGOT PASSWORD TYPES ---

type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// --- FORGOT PASSWORD HANDLER ---

// ForgotPassword
func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) error {
	// Get JSON
	req, err := httpio.BindJSON[ForgotPasswordRequest](w, r, h.validator)
	if err != nil {
		return err
	}

	if err := h.service.ForgotPassword(r.Context(), req.Email); err != nil {
		return err
	}

	// Success response is generic to prevent email enumeration
	httpio.RespondOK(w, r, struct{}{}, MsgForgotPasswordSuccess)

	return nil
}

// --- FORGOT PASSWORD SERVICE ---

// ForgotPassword
func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	// Get user
	user, err := s.user.GetByEmail(ctx, email)
	if err != nil {
		if apperr.Is(err, apperr.CodeNotFound) {
			return nil
		}
		return err
	}

	// Generate token
	resetToken, err := s.token.GeneratePasswordReset(user.ID)
	if err != nil {
		return apperr.New(apperr.CodeInternal, apperr.CodeInternal.Title(), apperr.WithErr(err))
	}

	// Emit event
	err = worker.EmitEvent(ctx, s.store, worker.EventForgotPassword, worker.AuthForgotPasswordPayload{
		Email: email,
		Token: resetToken,
	})
	if err != nil {
		return err
	}

	return nil
}
