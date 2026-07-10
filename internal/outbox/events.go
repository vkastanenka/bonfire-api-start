package outbox

import (
	"bonfire-api/internal/repository"
	"context"
	"encoding/json"
	"fmt"
)

type Emitter interface {
	OutboxEventCreate(ctx context.Context, arg repository.OutboxEventCreateParams) (repository.OutboxEvent, error)
}

type Type string

const (
	EventAuthRegister           Type = "auth.register"
	EventAuthResendVerification Type = "auth.retry-verification"
	EventAuthForgotPassword     Type = "auth.forgot-password"
)

type RegisterPayload struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Token    string `json:"token"`
}

func EmitRegister(ctx context.Context, db Emitter, payload RegisterPayload) error {
	return emitEvent(ctx, db, EventAuthRegister, payload)
}

type ResendVerificationPayload struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Token    string `json:"token"`
}

func EmitResendVerification(ctx context.Context, db Emitter, payload ResendVerificationPayload) error {
	return emitEvent(ctx, db, EventAuthResendVerification, payload)
}

type ForgotPasswordPayload struct {
	Email string `json:"email"`
	Token string `json:"token"`
}

func EmitForgotPassword(ctx context.Context, db Emitter, payload ForgotPasswordPayload) error {
	return emitEvent(ctx, db, EventAuthForgotPassword, payload)
}

func emitEvent(ctx context.Context, db Emitter, eventType Type, payload any) error {
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("outbox: failed to marshal payload for %s: %w", eventType, err)
	}

	_, err = db.OutboxEventCreate(ctx, repository.OutboxEventCreateParams{
		EventType: string(eventType),
		Payload:   jsonBytes,
	})
	if err != nil {
		return repository.NewError(err, repository.ScopeOutboxEvent)
	}

	return nil
}
