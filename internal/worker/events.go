package worker

import (
	"bonfire-api/internal/repository"
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgtype"
)

// Event types defined as constants to avoid magic strings throughout the codebase
const (
	EventUserRegistered = "user.registered"
)

// EmitEvent handles serialization and insertion into the outbox.
// This is called by services within an existing database transaction (qtx).
func EmitEvent(ctx context.Context, qtx *repository.Queries, eventType string, payload any) error {
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	_, err = qtx.OutboxEventCreate(ctx, repository.OutboxEventCreateParams{
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
