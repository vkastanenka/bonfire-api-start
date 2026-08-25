package session

import "bonfire-api/internal/errs"

var (
	ErrSessionExpired = errs.Unauthenticated("Session has expired.").
				Reason("SESSION_EXPIRED")
	ErrSessionRevoked = errs.Unauthenticated("Session has been revoked.").
				Reason("SESSION_REVOKED")
	ErrSessionInvalid = errs.Unauthenticated("Session is invalid.").
				Reason("SESSION_INVALID")
)
