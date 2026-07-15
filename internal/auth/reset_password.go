package auth

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/cache"
	"bonfire-api/internal/crypto"
	"bonfire-api/internal/httpio"
	"context"
	"log/slog"
	"net/http"
)

const (
	errInvalidResetToken = "Invalid or expired reset token."
)

type ResetPasswordRequest struct {
	Token    string `json:"token" validate:"required"`
	Password string `json:"password" validate:"identity_password"`
}

func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) error {
	req, err := httpio.BindJSON[ResetPasswordRequest](w, r)
	if err != nil {
		return err
	}

	if err := h.service.ResetPassword(r.Context(), ResetPasswordParams{
		req.Token,
		req.Password,
	}); err != nil {
		return err
	}

	httpio.RespondNoContent(w)
	return nil
}

type ResetPasswordParams struct {
	Token    string
	Password string
}

func (s *Service) ResetPassword(ctx context.Context, p ResetPasswordParams) error {
	claims, err := s.token.VerifyPasswordReset(p.Token)
	if err != nil {
		return apperr.NewTokenExpired(err, "")
	}

	userRow, err := s.user.GetByID(ctx, claims.UserID)
	if err != nil {
		return err
	}

	hashedPasswordBytes, err := crypto.HashPassword(p.Password)
	if err != nil {
		return apperr.NewInternal(err, "")
	}

	persistCtx := context.WithoutCancel(ctx)

	_, err = s.user.UpdatePassword(persistCtx, claims.UserID, string(hashedPasswordBytes))
	if err != nil {
		return err
	}

	failureKey := cache.AuthLoginFailuresKey(userRow.Email)
	lockoutKey := cache.AuthLoginFailuresKey(userRow.Email)

	if err := s.cache.Delete(persistCtx, failureKey); err != nil {
		slog.WarnContext(persistCtx, "failed to clear login failures on password reset", "error", err, "email", userRow.Email)
	}
	if err := s.cache.Delete(persistCtx, lockoutKey); err != nil {
		slog.WarnContext(persistCtx, "failed to lift login lockout on password reset", "error", err, "email", userRow.Email)
	}

	return nil
}
