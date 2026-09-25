package session

import "bonfire-api/internal/pkg/errs"

func ErrSessionExpired() *errs.Error {
	return errs.Unauthenticated("Session has expired.").
		Reason("SESSION_EXPIRED")
}

func ErrSessionRevoked() *errs.Error {
	return errs.Unauthenticated("Session has been revoked.").
		Reason("SESSION_REVOKED")
}

func ErrSessionInvalid() *errs.Error {
	return errs.Unauthenticated("Session is invalid.").
		Reason("SESSION_INVALID")
}
