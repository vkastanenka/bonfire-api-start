package auth

import (
	"bonfire-api/internal/pkg/errs"
	"errors"
)

const (
	ErrReasonIDGenerateFailed       = "ID_GENERATE_FAILED"
	ErrReasonCredentialsInvalid     = "CREDENTIALS_INVALID"
	ErrReasonCooldownExceeded       = "COOLDOWN_EXCEEDED"
	ErrReasonSessionRevoked         = "SESSION_REVOKED"
	ErrReasonSessionExpired         = "SESSION_EXPIRED"
	ErrReasonSessionPayloadRequired = "SESSION_PAYLOAD_REQUIRED"
	ErrReasonTokenInvalid           = "TOKEN_INVALID"
)

func ErrIDGenerateFailed() *errs.Error {
	return errs.Internal("Failed to generate ID.").
		Reason(ErrReasonIDGenerateFailed)
}

func ErrCredentialsInvalid() *errs.Error {
	return errs.Unauthenticated("Invalid credentials.").
		Reason(ErrReasonCredentialsInvalid).
		FieldViolation("email", "Invalid email or password.", "INVALID_CREDENTIALS").
		FieldViolation("password", "Invalid email or password.", "INVALID_CREDENTIALS")
}

func ErrCooldownExceeded() *errs.Error {
	return errs.ResourceExhausted("Too many attempts. Please try again soon.").
		Reason(ErrReasonCooldownExceeded)
}

func ErrSessionRevoked() *errs.Error {
	return errs.Unauthenticated("Session has been revoked.").
		Reason(ErrReasonSessionRevoked)
}

func ErrSessionExpired() *errs.Error {
	return errs.Unauthenticated("Session has expired.").
		Reason(ErrReasonSessionExpired)
}

func ErrTokenInvalid() *errs.Error {
	return errs.Unauthenticated("Invalid or expired token.").
		Reason(ErrReasonTokenInvalid).
		FieldViolation("token", "Invalid or expired token.", "INVALID")
}

func ErrSessionPayloadRequired() *errs.Error {
	return errs.Internal("Session payload required.").
		Reason(ErrReasonSessionExpired)
}

func ErrRefreshTokenInvalidReuse() *errs.Error {
	return errs.Unauthenticated("Invalid refresh token.").
		FieldViolation("refresh_token", "Token reuse detected.", "TOKEN_REUSE").
		Wrap(errors.New("refresh token reuse detected"))
}
