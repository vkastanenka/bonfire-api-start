package crypto

import "bonfire-api/internal/pkg/errs"

const (
	ErrReasonHashPasswordFailed = "HASH_PASSWORD_FAILED"
	ErrReasonInvalidCredentials = "INVALID_CREDENTIALS"
)

// ErrHashPasswordFailed returns when the hashing algorithm fails during password execution.
func ErrHashPasswordFailed() *errs.Error {
	return errs.Internal("Failed to hash password.").
		Reason(ErrReasonHashPasswordFailed)
}

// ErrInvalidCredentials returns when a password verification attempt fails or does not match a hash.
func ErrInvalidCredentials() *errs.Error {
	return errs.Unauthenticated("Invalid credentials.").
		Reason(ErrReasonInvalidCredentials)
}
