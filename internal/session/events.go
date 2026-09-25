package session

import (
	"bonfire-api/internal/outbox"
	"context"
	"time"

	"github.com/google/uuid"
)

type Worker interface {
	RegisterHandler(eventType string, handler outbox.Handler)
}

type Broadcaster interface {
	BroadcastToUser(ctx context.Context, recipientID uuid.UUID, eventType string, payload any, excludeSessionIDs ...uuid.UUID) error
	BroadcastToUserSession(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID, eventType string, payload any) error
}

const (
	EventRevoked    = "session.revoked"
	EventRevokedAll = "session.revoked_all"
)

func RegisterEvents(w Worker, b Broadcaster) {
	w.RegisterHandler(EventRevoked, newRevokedEventHandler(b))
	w.RegisterHandler(EventRevokedAll, newRevokedAllEventHandler(b))
}

type EventRevokedPayload struct {
	UserID    uuid.UUID `json:"userId"`
	SessionID uuid.UUID `json:"sessionId"`
	RevokedAt time.Time `json:"revokedAt"`
}

func newRevokedEventHandler(b Broadcaster) outbox.Handler {
	return outbox.BindHandler(func(ctx context.Context, m outbox.Metadata, p EventRevokedPayload) error {
		return b.BroadcastToUserSession(ctx, p.UserID, p.SessionID, EventRevoked, p)
	})
}

type EventRevokedAllPayload struct {
	UserID     uuid.UUID   `json:"userId"`
	SessionIDs []uuid.UUID `json:"sessionIds"`
	RevokedAt  time.Time   `json:"revokedAt"`
}

func newRevokedAllEventHandler(b Broadcaster) outbox.Handler {
	return outbox.BindHandler(func(ctx context.Context, m outbox.Metadata, p EventRevokedAllPayload) error {
		return b.BroadcastToUser(ctx, p.UserID, EventRevokedAll, p)
	})
}
