package auth

import (
	"bonfire-api/internal/errs"
	"errors"
)

func ErrEmailInvalid() error {
	return errs.InvalidArgument("Invalid email address.").
		FieldViolation("email", "Must be a valid email address.", "INVALID_EMAIL").
		Wrap(errors.New("invalid email address"))
}

func ErrCredentialsInvalid() error {
	return errs.Unauthenticated("Invalid email or password.").
		FieldViolation("email", "Invalid email or password.", "INVALID_CREDENTIALS").
		FieldViolation("password", "Invalid email or password.", "INVALID_CREDENTIALS").
		Wrap(errors.New("invalid credentials"))
}

func ErrRefreshTokenRequired() error {
	return errs.Unauthenticated("Missing refresh token.").
		FieldViolation("refresh_token", "Refresh token is required.", "REQUIRED").
		Wrap(errors.New("missing refresh token"))
}

func ErrRefreshTokenInvalid() error {
	return errs.Unauthenticated("Invalid or expired refresh token.").
		FieldViolation("refresh_token", "Invalid or expired refresh token.", "INVALID_TOKEN")
}

func ErrRefreshTokenInvalidReuse() error {
	return errs.Unauthenticated("Invalid refresh token.").
		FieldViolation("refresh_token", "Token reuse detected.", "TOKEN_REUSE").
		Wrap(errors.New("refresh token reuse detected"))
}

func ErrSessionRevoked() error {
	return errs.Unauthenticated("Session has been revoked.").
		FieldViolation("refresh_token", "Session has been revoked.", "SESSION_REVOKED").
		Wrap(errors.New("session revoked"))
}

func ErrSessionExpired() error {
	return errs.Unauthenticated("Session has expired.").
		FieldViolation("refresh_token", "Session has expired.", "SESSION_EXPIRED").
		Wrap(errors.New("session expired"))
}
