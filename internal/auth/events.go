package auth

import (
	"bonfire-api/internal/email"
	"bonfire-api/internal/fields"
	"bonfire-api/internal/outbox"
	"context"
	"encoding/json"
)

const (
	EventForgotPassword     = "auth.forgot_password"
	EventRegister           = "auth.register"
	EventResendVerification = "auth.retry_verification"
)

type EventForgotPasswordPayload struct {
	Email string `json:"email"`
	Token string `json:"token"`
}

type EventRegisterPayload struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Token    string `json:"token"`
}

type EventResendVerificationPayload struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Token    string `json:"token"`
}

func NewForgotPasswordOutboxHandler(mailer email.Mailer) outbox.Handler {
	return func(ctx context.Context, payload json.RawMessage) error {
		p, err := fields.ParseRawJSON[EventForgotPasswordPayload](payload)
		if err != nil {
			return err
		}
		return mailer.SendPasswordResetEmail(ctx, p.Email, p.Token)
	}
}

func NewRegisterOutboxHandler(mailer email.Mailer) outbox.Handler {
	return func(ctx context.Context, payload json.RawMessage) error {
		p, err := fields.ParseRawJSON[EventRegisterPayload](payload)
		if err != nil {
			return err
		}
		return mailer.SendRegisterEmail(ctx, p.Email, p.Username, p.Token)
	}
}

func NewResendVerificationOutboxHandler(mailer email.Mailer) outbox.Handler {
	return func(ctx context.Context, payload json.RawMessage) error {
		p, err := fields.ParseRawJSON[EventResendVerificationPayload](payload)
		if err != nil {
			return err
		}
		return mailer.SendResendVerificationEmail(ctx, p.Email, p.Username, p.Token)
	}
}

func RegisterOutboxHandlers(w *outbox.Worker, mailer email.Mailer) {
	w.RegisterHandler(EventRegister, NewRegisterOutboxHandler(mailer))
	w.RegisterHandler(EventResendVerification, NewResendVerificationOutboxHandler(mailer))
	w.RegisterHandler(EventForgotPassword, NewForgotPasswordOutboxHandler(mailer))
}
