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

// --- FORGOT PASSWORD TYPES ---

type ForgotPasswordReq struct {
	Email string `json:"email" validate:"auth_email"`
}

func (r *ForgotPasswordReq) Sanitize() {
	r.Email = sanitize.SanitizeEmail(r.Email)
}

// --- FORGOT PASSWORD CONSTANTS ---

// Messages
const (
	msgForgotPasswordSuccess = "Forgot password flow started. Please check your email for next steps."
)

// Values
const (
	forgotPasswordTimingWindow = 35 * time.Millisecond
	forgotPasswordCooldown     = 1 * time.Minute
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
	// Timing attacks defense
	defer crypto.ConstantWindow(forgotPasswordTimingWindow)()

	// Check cooldown
	cooldownKey := cache.ForgotPasswordCooldownKey(email)
	onCooldown, err := s.cache.Exists(ctx, cooldownKey)
	if err != nil {
		// Fail-Open
		slog.ErrorContext(ctx, "forgot password cooldown lookup failed", "error", err, "email", email)
	} else if onCooldown {
		// Exit silently
		return nil
	}

	// Get user
	userAuth, err := s.user.GetAuthByEmail(ctx, email)
	if err != nil {
		// Respond ok if not found
		if apperr.IsNotFound(err) {
			return nil
		}
		return err
	}

	// Allow only active users
	if !userAuth.IsActive() {
		return nil
	}

	// Generate token
	resetToken, err := s.token.GeneratePasswordReset(userAuth.ID, userAuth.SecurityVersion)
	if err != nil {
		return apperr.NewInternal(err)
	}

	// Emit event
	if err := worker.EmitForgotPassword(ctx, s.store, worker.ForgotPasswordPayload{
		Email: email,
		Token: resetToken,
	}); err != nil {
		return err
	}

	// Generate lock context to ensure cache write
	lockCtx := context.WithoutCancel(ctx)

	// Set cooldown
	if err := s.cache.Set(lockCtx, cooldownKey, true, forgotPasswordCooldown); err != nil {
		// Fail-Open
		slog.WarnContext(ctx, "failed to set forgot password cooldown", "error", err, "email", email)
	}

	return nil
}
