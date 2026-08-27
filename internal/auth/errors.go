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

func ErrUsernameInvalid() error {
	return errs.InvalidArgument("Invalid username.").
		FieldViolation("username", "Must be a valid username.", "INVALID_USERNAME").
		Wrap(errors.New("invalid username"))
}

func ErrDisplayNameInvalid(err error) error {
	return errs.InvalidArgument("Invalid display name format.").
		FieldViolation("display_name", "Invalid display name format.", "INVALID_DISPLAY_NAME").
		Wrap(err)
}

func ErrCredentialsInvalid() error {
	return errs.Unauthenticated("Invalid email or password.").
		FieldViolation("email", "Invalid email or password.", "INVALID_CREDENTIALS").
		FieldViolation("password", "Invalid email or password.", "INVALID_CREDENTIALS").
		Wrap(errors.New("invalid credentials"))
}

func ErrAccountLocked() error {
	return errs.PermissionDenied("Account is temporarily locked due to too many failed login attempts.").
		Wrap(errors.New("account locked"))
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

func ErrConflict(emailAvailable, usernameAvailable bool) error {
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

// errs.InvalidArgument("Reset token is required.").
// 			FieldViolation("token", "Reset token is required.", "REQUIRED").
// 			Wrap(errors.New("reset token is required"))

func ErrResetTokenRequired() error {
	return errs.InvalidArgument("Reset token is required.").
		FieldViolation("token", "Reset token is required.", "REQUIRED").
		Wrap(errors.New("reset token is required"))
}

func ErrResetTokenInvalid(err error) error {
	return errs.Unauthenticated("Invalid or expired reset token.").
		FieldViolation("token", "Invalid or expired reset token.", "INVALID_TOKEN").
		Wrap(err)
}

func ErrResetTokenUserNotFound(err error) error {
	return errs.Unauthenticated("Invalid or expired reset token.").
		FieldViolation("token", "User associated with this token no longer exists.", "USER_NOT_FOUND").
		Wrap(err)
}

func ErrPasswordInvalid(err error) error {
	return errs.InvalidArgument("Invalid password.").
		FieldViolation("password", "Password does not meet validation criteria.", "INVALID_PASSWORD").
		Wrap(err)
}

func ErrVerificationTokenRequired() error {
	return errs.InvalidArgument("Verification token is required.").
		FieldViolation("token", "Verification token is required.", "REQUIRED").
		Wrap(errors.New("verification token is required"))
}

func ErrVerificationTokenInvalid(err error) error {
	return errs.Unauthenticated("Invalid or expired verification token.").
		FieldViolation("token", "Invalid or expired verification token.", "INVALID_TOKEN").
		Wrap(err)
}

func ErrVerificationTokenUsed() error {
	return errs.Unauthenticated("Verification token has already been used.").
		FieldViolation("token", "Verification token has already been used.", "TOKEN_ALREADY_USED").
		Wrap(errors.New("verification token already used"))
}

func ErrVerificationTokenUserNotFound(err error) error {
	return errs.Unauthenticated("Invalid or expired verification token.").
		FieldViolation("token", "User associated with this token no longer exists.", "USER_NOT_FOUND").
		Wrap(err)
}