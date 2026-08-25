package auth

import (
	"bonfire-api/internal/email"
	"bonfire-api/internal/outbox"
	"context"
	"encoding/json"
	"fmt"
)

const (
	EventForgotPassword     = "auth.forgot-password"
	EventRegister           = "auth.register"
	EventResendVerification = "auth.retry-verification"
)

type ForgotPasswordPayload struct {
	Email string `json:"email"`
	Token string `json:"token"`
}

type RegisterPayload struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Token    string `json:"token"`
}

type ResendVerificationPayload struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Token    string `json:"token"`
}

// RegisterOutboxHandlers wires up all auth domain event handlers to the outbox worker.
func RegisterOutboxHandlers(w *outbox.Worker, mailer email.Mailer) {
	w.RegisterHandler(EventRegister, func(ctx context.Context, raw json.RawMessage) error {
		var p RegisterPayload
		if err := json.Unmarshal(raw, &p); err != nil {
			return fmt.Errorf("%w: malformed register payload: %v", outbox.ErrFatal, err)
		}
		return mailer.SendRegisterEmail(ctx, p.Email, p.Username, p.Token)
	})

	w.RegisterHandler(EventResendVerification, func(ctx context.Context, raw json.RawMessage) error {
		var p ResendVerificationPayload
		if err := json.Unmarshal(raw, &p); err != nil {
			return fmt.Errorf("%w: malformed resend verification payload: %v", outbox.ErrFatal, err)
		}
		return mailer.SendResendVerificationEmail(ctx, p.Email, p.Username, p.Token)
	})

	w.RegisterHandler(EventForgotPassword, func(ctx context.Context, raw json.RawMessage) error {
		var p ForgotPasswordPayload
		if err := json.Unmarshal(raw, &p); err != nil {
			return fmt.Errorf("%w: malformed forgot password payload: %v", outbox.ErrFatal, err)
		}
		return mailer.SendPasswordResetEmail(ctx, p.Email, p.Token)
	})
}
