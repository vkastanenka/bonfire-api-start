package crypto

import "bonfire-api/internal/pkg/errs"

const (
	ErrReasonHashPasswordFailed = "HASH_PASSWORD_FAILED"
	ErrReasonInvalidCredentials = "INVALID_CREDENTIALS"
)

// ErrHashPasswordFailed returns an Internal error when password hashing fails.
func ErrHashPasswordFailed() *errs.Error {
	return errs.Internal("Failed to hash password.").
		Reason(ErrReasonHashPasswordFailed)
}

// ErrInvalidCredentials returns an Unauthenticated error when a password comparison fails.
func ErrInvalidCredentials() *errs.Error {
	return errs.Unauthenticated("Invalid credentials.").
		Reason(ErrReasonInvalidCredentials)
}
