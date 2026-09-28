package auth

import (
	"bonfire-api/internal/outbox"
	"context"
	"time"
)

type Worker interface {
	RegisterHandler(eventType string, handler outbox.Handler)
}

type Mailer interface {
	SendPasswordReset(ctx context.Context, emailAddress string, resetToken string, requestedAt time.Time) error
	SendRegister(ctx context.Context, emailAddress string, username string, token string, requestedAt time.Time) error
	SendResendVerification(ctx context.Context, emailAddress string, username string, token string, requestedAt time.Time) error
}

const (
	EventPasswordResetRequested = "auth.password_reset_requested"
	EventRegistered             = "auth.user_registered"
	EventVerificationResent     = "auth.verification_resent"
)

func RegisterEvents(w Worker, m Mailer) {
	w.RegisterHandler(EventPasswordResetRequested, newPasswordResetRequestedHandler(m))
	w.RegisterHandler(EventRegistered, newRegisteredHandler(m))
	w.RegisterHandler(EventVerificationResent, newVerificationResentHandler(m))
}

type EventPasswordResetRequestedPayload struct {
	Email       string    `json:"email"`
	Token       string    `json:"token"`
	RequestedAt time.Time `json:"requestedAt"`
}

func newPasswordResetRequestedHandler(mailer Mailer) outbox.Handler {
	return outbox.BindHandler(func(ctx context.Context, m outbox.Metadata, p EventPasswordResetRequestedPayload) error {
		return mailer.SendPasswordReset(ctx, p.Email, p.Token, p.RequestedAt)
	})
}

type EventRegisteredPayload struct {
	Email       string    `json:"email"`
	Username    string    `json:"username"`
	Token       string    `json:"token"`
	RequestedAt time.Time `json:"requestedAt"`
}

func newRegisteredHandler(mailer Mailer) outbox.Handler {
	return outbox.BindHandler(func(ctx context.Context, m outbox.Metadata, p EventRegisteredPayload) error {
		return mailer.SendRegister(ctx, p.Email, p.Username, p.Token, p.RequestedAt)
	})
}

type EventVerificationResentPayload struct {
	Email       string    `json:"email"`
	Username    string    `json:"username"`
	Token       string    `json:"token"`
	RequestedAt time.Time `json:"requestedAt"`
}

func newVerificationResentHandler(mailer Mailer) outbox.Handler {
	return outbox.BindHandler(func(ctx context.Context, m outbox.Metadata, p EventVerificationResentPayload) error {
		return mailer.SendPasswordReset(ctx, p.Email, p.Token, p.RequestedAt)
	})
}
