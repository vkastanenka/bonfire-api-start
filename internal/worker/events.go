package worker

import (
	"bonfire-api/internal/repository"
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgtype"
)

// Define a mini-interface for the outbox capability
type OutboxExt interface {
	OutboxEventCreate(ctx context.Context, arg repository.OutboxEventCreateParams) (repository.OutboxEvent, error)
}

// Event types defined as constants to avoid magic strings throughout the codebase
const (
	EventUserRegistered = "user.registered"
	EventForgotPassword = "user.forgot-password"
)

// EmitEvent handles serialization and insertion into the outbox.
// This is called by services within an existing database transaction (qtx).
func EmitEvent(ctx context.Context, db OutboxExt, eventType string, payload any) error {
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	_, err = db.OutboxEventCreate(ctx, repository.OutboxEventCreateParams{
		EventType: eventType,
		Payload:   jsonBytes,
	})
	return err
}

// --- Payload Definitions ---

type AuthRegisterEventPayload struct {
	UserID pgtype.UUID `json:"user_id"`
}

type AuthForgotPasswordPayload struct {
	Email string `json:"email"`
	Token string `json:"token"`
}
