package worker

import (
	"bonfire-api/internal/repository"
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgtype"
)

// --- EVENT TYPES ---

type RegisterEventPayload struct {
	UserID pgtype.UUID `json:"user_id"`
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
	eventUserRegistered = "user.registered"
	eventForgotPassword = "user.forgot-password"
)

// --- EVENT FUNCTIONS ---

// EmitUserRegister
func EmitUserRegister(ctx context.Context, db OutboxExt, payload RegisterEventPayload) error {
	return emitEvent(ctx, db, eventUserRegistered, payload)
}

// EmitForgotPassword
func EmitForgotPassword(ctx context.Context, db OutboxExt, payload ForgotPasswordPayload) error {
	return emitEvent(ctx, db, eventForgotPassword, payload)
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
