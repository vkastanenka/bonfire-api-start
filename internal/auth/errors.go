package auth

import (
	"bonfire-api/internal/pkg/errs"
)

const (
	ErrReasonCooldownExceeded       = "COOLDOWN_EXCEEDED"
	ErrReasonCredentialsInvalid     = "CREDENTIALS_INVALID"
	ErrReasonIDGenerateFailed       = "ID_GENERATE_FAILED"
	ErrReasonSessionExpired         = "SESSION_EXPIRED"
	ErrReasonSessionNotFound        = "SESSION_NOT_FOUND"
	ErrReasonSessionPayloadRequired = "SESSION_PAYLOAD_REQUIRED"
	ErrReasonSessionRevoked         = "SESSION_REVOKED"
	ErrReasonTokenInvalid           = "TOKEN_INVALID"
)

func ErrCooldownExceeded() *errs.Error {
	return errs.ResourceExhausted("Too many attempts. Please try again soon.").
		Reason(ErrReasonCooldownExceeded)
}

func ErrCredentialsInvalid() *errs.Error {
	return errs.Unauthenticated("Invalid credentials.").
		Reason(ErrReasonCredentialsInvalid).
		FieldViolation("email", "Invalid email or password.", "INVALID_CREDENTIALS").
		FieldViolation("password", "Invalid email or password.", "INVALID_CREDENTIALS")
}

func ErrIDGenerateFailed() *errs.Error {
	return errs.Internal("Failed to generate ID.").
		Reason(ErrReasonIDGenerateFailed)
}

func ErrSessionExpired() *errs.Error {
	return errs.Unauthenticated("Session has expired.").
		Reason(ErrReasonSessionExpired)
}

func ErrSessionNotFound() *errs.Error {
	return errs.NotFound("Session not found.").
		Reason(ErrReasonSessionNotFound)
}

func ErrSessionPayloadRequired() *errs.Error {
	return errs.Internal("Session payload required.").
		Reason(ErrReasonSessionPayloadRequired)
}

func ErrSessionRevoked() *errs.Error {
	return errs.Unauthenticated("Session has been revoked.").
		Reason(ErrReasonSessionRevoked)
}

func ErrTokenInvalid() *errs.Error {
	return errs.Unauthenticated("Invalid or expired token.").
		Reason(ErrReasonTokenInvalid)
}
