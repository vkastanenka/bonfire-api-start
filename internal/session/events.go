package session

import (
	"time"

	"github.com/google/uuid"
)

const (
	EventRevoke    = "session.revoke"
	EventRevokeAll = "session.revoke-all"
)

type EventRevokePayload struct {
	SessionID uuid.UUID `json:"session_id"`
	UserID    uuid.UUID `json:"user_id"`
	RevokedAt time.Time `json:"revoked_at"`
}

type EventRevokeAllPayload struct {
	UserID     uuid.UUID   `json:"user_id"`
	SessionIDs []uuid.UUID `json:"session_ids"`
	RevokedAt  time.Time   `json:"revoked_at"`
}

// func NewRevokeOutboxHandler(gw outbox.Broadcaster) outbox.Handler {
// 	return func(ctx context.Context, payload json.RawMessage) error {
// 		p, err := fields.ParseRawJSON[EventRevokePayload](payload)
// 		if err != nil {
// 			return err
// 		}
// 		return outbox.NewSessionHandler(gw, EventRevoke, p.UserID, p.UserID, p.SessionID)(ctx, payload)
// 	}
// }

// func NewRevokeAllOutboxHandler(gw outbox.Broadcaster) outbox.Handler {
// 	return func(ctx context.Context, payload json.RawMessage) error {
// 		p, err := fields.ParseRawJSON[EventRevokeAllPayload](payload)
// 		if err != nil {
// 			return err
// 		}
// 		return outbox.NewUserHandler(gw, EventRevokeAll, p.UserID, p.UserID)(ctx, payload)
// 	}
// }
