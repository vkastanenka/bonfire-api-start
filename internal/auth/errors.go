package auth

import (
	"bonfire-api/internal/pkg/errs"
	"errors"
)

func ErrEmailInvalid() *errs.Error {
	return errs.InvalidArgument("Invalid email address.").
		FieldViolation("email", "Must be a valid email address.", "INVALID_EMAIL").
		Wrap(errors.New("invalid email address"))
}

func ErrUsernameInvalid() *errs.Error {
	return errs.InvalidArgument("Invalid username.").
		FieldViolation("username", "Must be a valid username.", "INVALID_USERNAME").
		Wrap(errors.New("invalid username"))
}

func ErrDisplayNameInvalid() *errs.Error {
	return errs.InvalidArgument("Invalid display name format.").
		FieldViolation("display_name", "Invalid display name format.", "INVALID_DISPLAY_NAME")
}

func ErrCredentialsInvalid() *errs.Error {
	return errs.Unauthenticated("Invalid email or password.").
		FieldViolation("email", "Invalid email or password.", "INVALID_CREDENTIALS").
		FieldViolation("password", "Invalid email or password.", "INVALID_CREDENTIALS").
		Wrap(errors.New("invalid credentials"))
}

func ErrAccountLocked() *errs.Error {
	return errs.PermissionDenied("Account is temporarily locked due to too many failed login attempts.").
		Wrap(errors.New("account locked"))
}

func ErrRefreshTokenRequired() *errs.Error {
	return errs.Unauthenticated("Missing refresh token.").
		FieldViolation("refresh_token", "Refresh token is required.", "REQUIRED").
		Wrap(errors.New("missing refresh token"))
}

func ErrRefreshTokenInvalid() *errs.Error {
	return errs.Unauthenticated("Invalid or expired refresh token.").
		FieldViolation("refresh_token", "Invalid or expired refresh token.", "INVALID_TOKEN")
}

func ErrRefreshTokenInvalidReuse() *errs.Error {
	return errs.Unauthenticated("Invalid refresh token.").
		FieldViolation("refresh_token", "Token reuse detected.", "TOKEN_REUSE").
		Wrap(errors.New("refresh token reuse detected"))
}

func ErrSessionRevoked() *errs.Error {
	return errs.Unauthenticated("Session has been revoked.").
		FieldViolation("refresh_token", "Session has been revoked.", "SESSION_REVOKED").
		Wrap(errors.New("session revoked"))
}

func ErrSessionExpired() *errs.Error {
	return errs.Unauthenticated("Session has expired.").
		FieldViolation("refresh_token", "Session has expired.", "SESSION_EXPIRED").
		Wrap(errors.New("session expired"))
}

func ErrConflict(emailAvailable, usernameAvailable bool) *errs.Error {
	e := errs.AlreadyExists("The provided email or username is already taken.").
		Wrap(errors.New("registration conflict"))

	if !emailAvailable {
		e = e.FieldViolation("email", "This email address is already registered.", "ALREADY_EXISTS")
	}
	if !usernameAvailable {
		e = e.FieldViolation("username", "This username is already taken.", "ALREADY_EXISTS")
	}

	return e
}

func ErrResetTokenRequired() *errs.Error {
	return errs.InvalidArgument("Reset token is required.").
		FieldViolation("token", "Reset token is required.", "REQUIRED").
		Wrap(errors.New("reset token is required"))
}

func ErrResetTokenInvalid() *errs.Error {
	return errs.Unauthenticated("Invalid or expired reset token.").
		FieldViolation("token", "Invalid or expired reset token.", "INVALID_TOKEN")
}

func ErrResetTokenUserNotFound() *errs.Error {
	return errs.Unauthenticated("Invalid or expired reset token.").
		FieldViolation("token", "User associated with this token no longer exists.", "USER_NOT_FOUND")
}

func ErrPasswordInvalid() *errs.Error {
	return errs.InvalidArgument("Invalid password.").
		FieldViolation("password", "Password does not meet validation criteria.", "INVALID_PASSWORD")
}

func ErrVerificationTokenRequired() *errs.Error {
	return errs.InvalidArgument("Verification token is required.").
		FieldViolation("token", "Verification token is required.", "REQUIRED").
		Wrap(errors.New("verification token is required"))
}

func ErrVerificationTokenInvalid() *errs.Error {
	return errs.Unauthenticated("Invalid or expired verification token.").
		FieldViolation("token", "Invalid or expired verification token.", "INVALID_TOKEN")
}

func ErrVerificationTokenUsed() *errs.Error {
	return errs.Unauthenticated("Verification token has already been used.").
		FieldViolation("token", "Verification token has already been used.", "TOKEN_ALREADY_USED").
		Wrap(errors.New("verification token already used"))
}

func ErrVerificationTokenUserNotFound() *errs.Error {
	return errs.Unauthenticated("Invalid or expired verification token.").
		FieldViolation("token", "User associated with this token no longer exists.", "USER_NOT_FOUND")
}
