package worker

import (
	"bonfire-api/internal/repository"
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgtype"
)

type OutboxExt interface {
	OutboxEventCreate(ctx context.Context, arg repository.OutboxEventCreateParams) (repository.OutboxEvent, error)
}

const (
	eventUserRegistered = "user.registered"
	eventForgotPassword = "user.forgot-password"
)

// --- TYPE-SAFE EXPORTED EMITTERS ---

// EmitUserRegister safely queues a user registration event.
func EmitUserRegister(ctx context.Context, db OutboxExt, payload RegisterEventPayload) error {
	return emitEvent(ctx, db, eventUserRegistered, payload)
}

// EmitForgotPassword safely queues a password reset intent event.
func EmitForgotPassword(ctx context.Context, db OutboxExt, payload ForgotPasswordPayload) error {
	return emitEvent(ctx, db, eventForgotPassword, payload)
}

// --- PRIVATE SERIALIZATION LAYER ---

// emitEvent is private so nobody can bypass the type-safe wrappers above.
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

// --- PAYLOAD DEFINITIONS ---

type RegisterEventPayload struct {
	UserID pgtype.UUID `json:"user_id"`
}

type ForgotPasswordPayload struct {
	Email string `json:"email"`
	Token string `json:"token"`
}
