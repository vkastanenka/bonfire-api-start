package session

import (
	"context"
	"encoding/json"

	"bonfire-api/internal/fields"
	"bonfire-api/internal/outbox"
)

const (
	EventSessionRevoke    = "session.revoke"
	EventSessionRevokeAll = "session.revoke-all"
)

type EventSessionRevokePayload struct {
	SessionID string `json:"session_id"`
	UserID    string `json:"user_id"`
	RevokedAt string `json:"revoked_at"`
}

type EventSessionRevokeAllPayload struct {
	UserID     string   `json:"user_id"`
	SessionIDs []string `json:"session_ids"`
	RevokedAt  string   `json:"revoked_at"`
}

func NewRevokeOutboxHandler(gw outbox.Broadcaster) outbox.Handler {
	return func(ctx context.Context, payload json.RawMessage) error {
		p, err := fields.ParseRawJSON[EventSessionRevokePayload](payload)
		if err != nil {
			return err
		}
		return outbox.NewSessionHandler(gw, EventSessionRevoke, p.UserID, p.UserID, p.SessionID)(ctx, payload)
	}
}

func NewRevokeAllOutboxHandler(gw outbox.Broadcaster) outbox.Handler {
	return func(ctx context.Context, payload json.RawMessage) error {
		p, err := fields.ParseRawJSON[EventSessionRevokeAllPayload](payload)
		if err != nil {
			return err
		}
		return outbox.NewUserHandler(gw, EventSessionRevokeAll, p.UserID, p.UserID)(ctx, payload)
	}
}
