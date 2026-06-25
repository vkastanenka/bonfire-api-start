package auth

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/cache"
	"bonfire-api/internal/crypto"
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/sanitize"
	"bonfire-api/internal/worker"
	"context"
	"log/slog"
	"net/http"
	"time"
)

// --- RESEND VERIFICATION EMAIL TYPES ---

type ResendVerificationEmailReq struct {
	Email string `json:"email" validate:"identity.email"`
}

func (r *ResendVerificationEmailReq) Sanitize() {
	r.Email = sanitize.SanitizeEmail(r.Email)
}

// --- RESEND VERIFICATION EMAIL CONSTANTS ---

// Messages
const (
	msgResendVerificationEmailSuccess = "resend_verification_email_success"
)

// Values
const (
	resendVerificationEmailTimingWindow = 35 * time.Millisecond
	resendVerificationEmailCooldown     = 1 * time.Minute
)

// --- RESEND VERIFICATION EMAIL HANDLER ---

func (h *Handler) ResendVerificationEmail(w http.ResponseWriter, r *http.Request) error {
	// Get JSON
	req, err := httpio.BindJSON[ResendVerificationEmailReq](w, r, h.validator)
	if err != nil {
		return err
	}

	// Call service
	if err := h.service.ResendVerificationEmail(r.Context(), req.Email); err != nil {
		return err
	}

	// Respond
	httpio.RespondOK(w, r, struct{}{}, msgResendVerificationEmailSuccess)
	return nil
}

// --- RESEND VERIFICATION EMAIL SERVICE ---

func (s *Service) ResendVerificationEmail(ctx context.Context, email string) error {
	// Timing attacks defense
	defer crypto.ConstantWindow(resendVerificationEmailTimingWindow)()

	// Check cooldown
	cooldownKey := cache.ResendVerificationCooldownKey(email)
	onCooldown, err := s.cache.Exists(ctx, cooldownKey)
	if err != nil {
		slog.ErrorContext(ctx, "resend verification cooldown lookup failed", "error", err, "email", email)
	} else if onCooldown {
		return nil
	}

	// Get user auth
	userAuth, err := s.user.GetAuthByEmail(ctx, email)
	if err != nil {
		if apperr.IsNotFound(err) {
			return nil
		}
		return err
	}

	// Check if verified
	if userAuth.VerifiedAt != nil {
		return nil
	}

	// Generate fresh token
	token, err := s.token.GenerateVerification(userAuth.ID, userAuth.SecurityVersion)
	if err != nil {
		return apperr.NewInternal(err)
	}

	// Persistence boundary
	persistCtx := context.WithoutCancel(ctx)

	// Emit event
	if err := worker.EmitResendVerificationEmail(persistCtx, s.store, worker.ResendVerificationEmailPayload{
		Email:    userAuth.Email,
		Username: userAuth.Username,
		Token:    token,
	}); err != nil {
		return err
	}

	// Set cooldown
	if err := s.cache.Set(persistCtx, cooldownKey, true, resendVerificationEmailCooldown); err != nil {
		slog.WarnContext(persistCtx, "failed to set resend verification cooldown", "error", err, "email", email)
	}

	return nil
}
