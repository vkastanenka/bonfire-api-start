package auth

import (
	"context"
	"time"

	"bonfire-api/internal/outbox"

	"github.com/google/uuid"
)

// TODO: CONFIRM PAYLOAD VS META IMPLEMENTATION

const (
	EventPasswordResetRequested = "auth.password_reset_requested"
	EventRegistered             = "auth.user_registered"
	EventVerificationResent     = "auth.verification_resent"
	EventSessionRevoked         = "auth.session_revoked"
	EventSessionRevokedAll      = "auth.session_revoked_all"
)

func RegisterEvents(w Worker, m Mailer, b Broadcaster) {
	w.RegisterHandler(EventPasswordResetRequested, newPasswordResetRequestedHandler(m))
	w.RegisterHandler(EventRegistered, newRegisteredHandler(m))
	w.RegisterHandler(EventVerificationResent, newVerificationResentHandler(m))
	w.RegisterHandler(EventSessionRevoked, newSessionRevokedEventHandler(b))
	w.RegisterHandler(EventSessionRevokedAll, newSessionRevokedAllEventHandler(b))
}

// --- Password Reset ---

type EventPasswordResetRequestedPayload struct {
	Email       string    `json:"email"`
	Username    string    `json:"username"`
	Token       string    `json:"token"`
	RequestedAt time.Time `json:"requestedAt"`
}

func newEventPasswordResetRequestedPayload(email, username, token string, now time.Time) EventPasswordResetRequestedPayload {
	return EventPasswordResetRequestedPayload{
		Email:       email,
		Username:    username,
		Token:       token,
		RequestedAt: now,
	}
}

func newPasswordResetRequestedHandler(mailer Mailer) outbox.Handler {
	return outbox.BindHandler(func(ctx context.Context, m outbox.Metadata, p EventPasswordResetRequestedPayload) error {
		return mailer.SendPasswordReset(ctx, p.Email, p.Token, p.RequestedAt)
	})
}

// --- Registration ---

type EventRegisteredPayload struct {
	Email       string    `json:"email"`
	Username    string    `json:"username"`
	Token       string    `json:"token"`
	RequestedAt time.Time `json:"requestedAt"`
}

func newEventRegisteredPayload(email, username, token string, now time.Time) EventRegisteredPayload {
	return EventRegisteredPayload{
		Email:       email,
		Username:    username,
		Token:       token,
		RequestedAt: now,
	}
}

func newRegisteredHandler(mailer Mailer) outbox.Handler {
	return outbox.BindHandler(func(ctx context.Context, m outbox.Metadata, p EventRegisteredPayload) error {
		return mailer.SendRegister(ctx, p.Email, p.Username, p.Token, p.RequestedAt)
	})
}

// --- Verification Resent ---

type EventVerificationResentPayload struct {
	Email       string    `json:"email"`
	Username    string    `json:"username"`
	Token       string    `json:"token"`
	RequestedAt time.Time `json:"requestedAt"`
}

func newEventVerificationResentPayload(email, username, token string, now time.Time) EventVerificationResentPayload {
	return EventVerificationResentPayload{
		Email:       email,
		Username:    username,
		Token:       token,
		RequestedAt: now,
	}
}

func newVerificationResentHandler(mailer Mailer) outbox.Handler {
	return outbox.BindHandler(func(ctx context.Context, m outbox.Metadata, p EventVerificationResentPayload) error {
		return mailer.SendResendVerification(ctx, p.Email, p.Username, p.Token, p.RequestedAt)
	})
}

// --- Session Revoked ---

type EventSessionRevokedPayload struct {
	UserID    uuid.UUID `json:"userId"`
	SessionID uuid.UUID `json:"sessionId"`
	RevokedAt time.Time `json:"revokedAt"`
}

func newEventSessionRevokedPayload(userID, sessionID uuid.UUID, now time.Time) EventSessionRevokedPayload {
	return EventSessionRevokedPayload{
		UserID:    userID,
		SessionID: sessionID,
		RevokedAt: now,
	}
}

func newSessionRevokedEventHandler(b Broadcaster) outbox.Handler {
	return outbox.BindHandler(func(ctx context.Context, m outbox.Metadata, p EventSessionRevokedPayload) error {
		return b.BroadcastToUserSession(ctx, p.UserID, p.SessionID, EventSessionRevoked, p)
	})
}

// --- Session Revoked All ---

type EventSessionRevokedAllPayload struct {
	UserID     uuid.UUID   `json:"userId"`
	SessionIDs []uuid.UUID `json:"sessionIds"`
	RevokedAt  time.Time   `json:"revokedAt"`
}

func newEventSessionRevokedAllPayload(userID uuid.UUID, sessionIDs []uuid.UUID, now time.Time) EventSessionRevokedAllPayload {
	return EventSessionRevokedAllPayload{
		UserID:     userID,
		SessionIDs: sessionIDs,
		RevokedAt:  now,
	}
}

func newSessionRevokedAllEventHandler(b Broadcaster) outbox.Handler {
	return outbox.BindHandler(func(ctx context.Context, m outbox.Metadata, p EventSessionRevokedAllPayload) error {
		return b.BroadcastToUser(ctx, p.UserID, EventSessionRevokedAll, p)
	})
}
