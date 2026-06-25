package worker

import (
	"bonfire-api/internal/repository"
	"context"
	"encoding/json"
)

// --- EVENT TYPES ---

type RegisterPayload struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Token    string `json:"token"`
}

type ResendVerificationEmailPayload struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Token    string `json:"token"`
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
	eventAuthRegister           = "auth.register"
	eventAuthResendVerification = "auth.resend-verification"
	eventAuthForgotPassword     = "auth.forgot-password"
)

// --- EVENT FUNCTIONS ---

// EmitRegister
func EmitRegister(ctx context.Context, db OutboxExt, payload RegisterPayload) error {
	return emitEvent(ctx, db, eventAuthRegister, payload)
}

// EmitResendVerification
func EmitResendVerificationEmail(ctx context.Context, db OutboxExt, payload ResendVerificationEmailPayload) error {
	return emitEvent(ctx, db, eventAuthResendVerification, payload)
}

// EmitForgotPassword
func EmitForgotPassword(ctx context.Context, db OutboxExt, payload ForgotPasswordPayload) error {
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
