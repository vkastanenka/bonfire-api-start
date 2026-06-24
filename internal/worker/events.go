package worker

import (
	"bonfire-api/internal/repository"
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgtype"
)

// --- EVENT TYPES ---

type RegisterEventPayload struct {
	UserID   pgtype.UUID `json:"user_id"`
	Email    string      `json:"email"`
	Username string      `json:"username"`
	Token    string      `json:"token"`
}

type ForgotPasswordPayload struct {
	Email string `json:"email"`
	Token string `json:"token"`
}

type OutboxExt interface {
	OutboxEventCreate(ctx context.Context, arg repository.OutboxEventCreateParams) (repository.OutboxEvent, error)
}

// --- EVENT CONSTANTS ---

const (
	eventAuthRegister       = "auth.register"
	eventAuthForgotPassword = "auth.forgot-password"
)

// --- EVENT FUNCTIONS ---

// EmitAuthRegister
func EmitAuthRegister(ctx context.Context, db OutboxExt, payload RegisterEventPayload) error {
	return emitEvent(ctx, db, eventAuthRegister, payload)
}

// EmitAuthForgotPassword
func EmitAuthForgotPassword(ctx context.Context, db OutboxExt, payload ForgotPasswordPayload) error {
	return emitEvent(ctx, db, eventAuthForgotPassword, payload)
}

// --- EVENT HELPERS ---

// emitEvent
func emitEvent(ctx context.Context, db OutboxExt, eventType string, payload any) error {
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
