package auth

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/outbox"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
)

func (h *Handler) ResendVerify(w http.ResponseWriter, r *http.Request) error {
	userID, err := httpio.GetCtxUserID(r.Context())
	if err != nil {
		return err
	}

	if err := h.service.ResendVerify(r.Context(), userID); err != nil {
		return err
	}

	httpio.RespondNoContent(w)
	return nil
}

func AuthCooldownResendVerificationKey(userID uuid.UUID) string {
	return fmt.Sprintf("auth:cooldown:resend-verify:{%s}", userID.String())
}

func (s *Service) ResendVerify(ctx context.Context, userId uuid.UUID) error {
	cooldownKey := AuthCooldownResendVerificationKey(userId)
	onCooldown, err := s.cache.Exists(ctx, cooldownKey)
	if err != nil {
		slog.ErrorContext(ctx, "resend verification cooldown lookup failed", "error", err, "user_id", userId)
	} else if onCooldown {
		return nil
	}

	userRow, err := s.user.GetByID(ctx, userId)
	if err != nil {
		if apperr.IsNotFound(err) {
			return nil
		}
		return err
	}

	if userRow.IsVerified() {
		return nil
	}

	token, _, err := s.token.GenerateEmailVerify(userId)
	if err != nil {
		return apperr.NewInternal(err, "")
	}

	persistCtx := context.WithoutCancel(ctx)

	if err := outbox.EmitResendVerification(persistCtx, s.store, outbox.ResendVerificationPayload{
		Email:    userRow.Email,
		Username: userRow.Username,
		Token:    token,
	}); err != nil {
		return err
	}

	if err := s.cache.Set(persistCtx, cooldownKey, true, 1*time.Minute); err != nil {
		slog.WarnContext(persistCtx, "failed to set resend verification cooldown", "error", err, "user_id", userId)
	}

	return nil
}
