package auth

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/httpio"
	"context"
	"net/http"
)

// --- VERIFY EMAIL CONSTANTS ---

// Messages
const (
	MsgVerifyEmailSuccess = "verify_email_success"
)

// Errors
const (
	ErrInvalidRefreshToken = "The refresh token is invalid or expired."
)

// --- VERIFY EMAIL TYPES ---

type VerifyEmailReq struct {
	Token string `json:"token" validate:"required"`
}

// --- VERIFY EMAIL HANDLER ---

// VerifyEmail handles incoming verification tokens sent from the frontend client.
func (h *Handler) VerifyEmail(w http.ResponseWriter, r *http.Request) error {
	// Get JSON
	req, err := httpio.BindJSON[VerifyEmailReq](w, r, h.validator)
	if err != nil {
		return err
	}

	// Verify email
	if err := h.service.VerifyEmail(r.Context(), req.Token); err != nil {
		return err
	}

	// Respond
	httpio.RespondOK(w, r, struct{}{}, MsgVerifyEmailSuccess)

	return nil
}

// --- VERIFY EMAIL SERVICE ---

func (s *Service) VerifyEmail(ctx context.Context, tokenStr string) error {
	// Verify refresh token
	claims, err := s.token.VerifyRefresh(tokenStr)
	if err != nil {
		return apperr.New(apperr.CodeUnauthenticated, ErrInvalidRefreshToken, apperr.WithErr(err))
	}

	// Mark user verification
	_, err = s.user.MarkVerified(ctx, claims.UserID)
	if err != nil {
		return err
	}

	return nil
}
