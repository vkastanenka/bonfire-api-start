package crypto

import "bonfire-api/internal/pkg/errs"

const (
	ErrReasonPasswordHashFailed = "PASSWORD_HASH_FAILED"
	ErrReasonCredentialsInvalid = "CREDENTIALS_INVALID"
)

// ErrPasswordHashFailed returns an Internal error when the hashing algorithm fails during password execution.
func ErrPasswordHashFailed() *errs.Error {
	return errs.Internal("Failed to hash password.").
		Reason(ErrReasonPasswordHashFailed)
}

// ErrCredentialsInvalid returns an Unauthenticated error when a password verification attempt fails or does not match a hash.
func ErrCredentialsInvalid() *errs.Error {
	return errs.Unauthenticated("Invalid credentials.").
		Reason(ErrReasonCredentialsInvalid)
}
