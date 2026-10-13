package auth

import (
	"context"
	"time"

	"bonfire-api/internal/outbox"

	"github.com/google/uuid"
)

const (
	EventRegistered             = "auth.user_registered"
	EventPasswordResetRequested = "auth.password_reset_requested"
	EventVerificationResent     = "auth.verification_resent"
	EventSessionRevoked         = "auth.session_revoked"
	EventSessionRevokedAll      = "auth.session_revoked_all"
)

func RegisterEvents(w Worker, m Mailer, b Broadcaster) {
	w.RegisterHandler(EventRegistered, newRegisteredHandler(m))
	w.RegisterHandler(EventPasswordResetRequested, newPasswordResetRequestedHandler(m))
	w.RegisterHandler(EventVerificationResent, newVerificationResentHandler(m))
	w.RegisterHandler(EventSessionRevoked, newSessionRevokedEventHandler(b))
	w.RegisterHandler(EventSessionRevokedAll, newSessionRevokedAllEventHandler(b))
}

type EventRegisteredPayload struct {
	Email     string    `json:"email"`
	Username  string    `json:"username"`
	Token     string    `json:"token"`
	Timestamp time.Time `json:"timestamp"`
}

func newEventRegisteredPayload(email, username, token string, timestamp time.Time) EventRegisteredPayload {
	return EventRegisteredPayload{
		Email:     email,
		Username:  username,
		Token:     token,
		Timestamp: timestamp,
	}
}

func newRegisteredHandler(mailer Mailer) outbox.Handler {
	return outbox.BindHandler(func(ctx context.Context, m outbox.Metadata, p EventRegisteredPayload) error {
		return mailer.SendRegister(ctx, p.Email, p.Username, p.Token, p.Timestamp)
	})
}

type EventPasswordResetRequestedPayload struct {
	Email     string    `json:"email"`
	Username  string    `json:"username"`
	Token     string    `json:"token"`
	Timestamp time.Time `json:"timestamp"`
}

func newEventPasswordResetRequestedPayload(email, username, token string, timestamp time.Time) EventPasswordResetRequestedPayload {
	return EventPasswordResetRequestedPayload{
		Email:     email,
		Username:  username,
		Token:     token,
		Timestamp: timestamp,
	}
}

func newPasswordResetRequestedHandler(mailer Mailer) outbox.Handler {
	return outbox.BindHandler(func(ctx context.Context, m outbox.Metadata, p EventPasswordResetRequestedPayload) error {
		return mailer.SendPasswordReset(ctx, p.Email, p.Username, p.Token, p.Timestamp)
	})
}

type EventVerificationResentPayload struct {
	Email     string    `json:"email"`
	Username  string    `json:"username"`
	Token     string    `json:"token"`
	Timestamp time.Time `json:"timestamp"`
}

func newEventVerificationResentPayload(email, username, token string, timestamp time.Time) EventVerificationResentPayload {
	return EventVerificationResentPayload{
		Email:     email,
		Username:  username,
		Token:     token,
		Timestamp: timestamp,
	}
}

func newVerificationResentHandler(mailer Mailer) outbox.Handler {
	return outbox.BindHandler(func(ctx context.Context, m outbox.Metadata, p EventVerificationResentPayload) error {
		return mailer.SendResendVerification(ctx, p.Email, p.Username, p.Token, p.Timestamp)
	})
}

type EventSessionRevokedPayload struct {
	RevokedSessionID uuid.UUID `json:"revokedSessionID"`
}

func newEventSessionRevokedPayload(revokedSessionID uuid.UUID) EventSessionRevokedPayload {
	return EventSessionRevokedPayload{
		RevokedSessionID: revokedSessionID,
	}
}

func newSessionRevokedEventHandler(b Broadcaster) outbox.Handler {
	return outbox.BindHandler(func(ctx context.Context, m outbox.Metadata, p EventSessionRevokedPayload) error {
		return b.BroadcastToUserSession(ctx, *m.ActorID, p.RevokedSessionID, EventSessionRevoked, p)
	})
}

func newSessionRevokedAllEventHandler(b Broadcaster) outbox.Handler {
	return outbox.BindHandler(func(ctx context.Context, m outbox.Metadata, p struct{}) error {
		return b.BroadcastToUser(ctx, *m.ActorID, EventSessionRevokedAll, struct{}{})
	})
}
