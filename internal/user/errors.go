package user

import (
	"fmt"

	"bonfire-api/internal/errs"
)

func ErrBioTooLong(field string) *errs.Error {
	return errs.InvalidArgument("Bio is too long.").
		Reason("BIO_TOO_LONG").
		FieldViolation(field, fmt.Sprintf("Bio cannot exceed %d characters.", MaxBioLength), "TOO_LONG")
}

func ErrDisplayNameRequired(field string) *errs.Error {
	return errs.InvalidArgument("Display name is required.").
		Reason("DISPLAY_NAME_REQUIRED").
		FieldViolation(field, "Display name cannot be empty.", "REQUIRED")
}

func ErrDisplayNameTooLong(field string) *errs.Error {
	return errs.InvalidArgument("Display name is too long.").
		Reason("DISPLAY_NAME_TOO_LONG").
		FieldViolation(field, fmt.Sprintf("Display name cannot exceed %d characters.", MaxDisplayNameLength), "TOO_LONG")
}

func ErrEmailRequired(field string) *errs.Error {
	return errs.InvalidArgument("Email is required.").
		Reason("EMAIL_REQUIRED").
		FieldViolation(field, "Email cannot be empty.", "REQUIRED")
}

func ErrEmailTooLong(field string) *errs.Error {
	return errs.InvalidArgument("Email is too long.").
		Reason("EMAIL_TOO_LONG").
		FieldViolation(field, fmt.Sprintf("Email cannot exceed %d characters.", MaxEmailLength), "TOO_LONG")
}

func ErrEmailInvalid(field string) *errs.Error {
	return errs.InvalidArgument("Email format is invalid.").
		Reason("EMAIL_INVALID").
		FieldViolation(field, "Must be a valid email address.", "INVALID_FORMAT")
}

func ErrPasswordRequired(field string) *errs.Error {
	return errs.InvalidArgument("Password is required.").
		Reason("PASSWORD_REQUIRED").
		FieldViolation(field, "Password cannot be empty.", "REQUIRED")
}

func ErrPasswordTooShort(field string) *errs.Error {
	return errs.InvalidArgument("Password is too short.").
		Reason("PASSWORD_TOO_SHORT").
		FieldViolation(field, fmt.Sprintf("Password must be at least %d characters.", MinPasswordLength), "TOO_SHORT")
}

func ErrPasswordTooLong(field string) *errs.Error {
	return errs.InvalidArgument("Password is too long.").
		Reason("PASSWORD_TOO_LONG").
		FieldViolation(field, fmt.Sprintf("Password cannot exceed %d characters.", MaxPasswordLength), "TOO_LONG")
}

func ErrPasswordHashRequired(field string) *errs.Error {
	return errs.InvalidArgument("Password hash is required.").
		Reason("PASSWORD_HASH_REQUIRED").
		FieldViolation(field, "Password hash cannot be empty.", "REQUIRED")
}

func ErrPasswordHashTooShort(field string) *errs.Error {
	return errs.InvalidArgument("Password hash is too short.").
		Reason("PASSWORD_HASH_TOO_SHORT").
		FieldViolation(field, fmt.Sprintf("Password hash must be at least %d characters.", MinPasswordHashLength), "TOO_SHORT")
}

func ErrPasswordHashTooLong(field string) *errs.Error {
	return errs.InvalidArgument("Password hash is too long.").
		Reason("PASSWORD_HASH_TOO_LONG").
		FieldViolation(field, fmt.Sprintf("Password hash cannot exceed %d characters.", MaxPasswordHashLength), "TOO_LONG")
}

func ErrPhoneInvalid(field string) *errs.Error {
	return errs.InvalidArgument("Phone format is invalid.").
		Reason("PHONE_INVALID").
		FieldViolation(field, "Phone must be in international E.164 format (e.g., +1234567890).", "INVALID_FORMAT")
}

func ErrPreferredPresenceInvalid(field string) *errs.Error {
	return errs.InvalidArgument("Invalid preferred presence status.").
		Reason("PREFERRED_PRESENCE_INVALID").
		FieldViolation(field, "Must be one of: idle, busy, dnd.", "INVALID_ENUM_VALUE")
}

func ErrPreferredPresenceDurationInvalid() *errs.Error {
	return errs.InvalidArgument("Invalid preferred presence duration.").
		Reason("PREFERRED_PRESENCE_DURATION_INVALID").
		FieldViolation("preferred_presence_duration", "Must be one of: 15_MIN, 1_HOUR, 8_HOURS, 24_HOURS, 3_DAYS, FOREVER.", "INVALID_ENUM_VALUE").
		Meta("domain", "PREFERRED_PRESENCE_DURATION")
}

func ErrUsernameRequired(field string) *errs.Error {
	return errs.InvalidArgument("Username is required.").
		Reason("USERNAME_REQUIRED").
		FieldViolation(field, "Username cannot be empty.", "REQUIRED")
}

func ErrUsernameTooShort(field string) *errs.Error {
	return errs.InvalidArgument("Username is too short.").
		Reason("USERNAME_TOO_SHORT").
		FieldViolation(field, fmt.Sprintf("Username must be at least %d characters.", MinUsernameLength), "TOO_SHORT")
}

func ErrUsernameTooLong(field string) *errs.Error {
	return errs.InvalidArgument("Username is too long.").
		Reason("USERNAME_TOO_LONG").
		FieldViolation(field, fmt.Sprintf("Username cannot exceed %d characters.", MaxUsernameLength), "TOO_LONG")
}

func ErrUsernameInvalid(field string) *errs.Error {
	return errs.InvalidArgument("Username format is invalid.").
		Reason("USERNAME_INVALID").
		FieldViolation(field, "Username can only contain alphanumeric characters, dots, and underscores.", "INVALID_FORMAT")
}

func ErrUsernameReserved(field string) *errs.Error {
	return errs.InvalidArgument("Username is reserved.").
		Reason("USERNAME_RESERVED").
		FieldViolation(field, "This username is reserved and cannot be used.", "RESERVED_VALUE")
}

func ErrInvalidPassword(field string) *errs.Error {
	return errs.Unauthenticated("Invalid password.").
		Reason("INVALID_PASSWORD").
		FieldViolation(field, "Invalid password.", "INVALID_PASSWORD")
}

func ErrPasswordMismatch(field string) *errs.Error {
	return errs.InvalidArgument("Passwords must match.").
		Reason("PASSWORD_MISMATCH").
		FieldViolation(field, "Passwords do not match.", "PASSWORD_MISMATCH")
}

func ErrPasswordHashFailed(err error) *errs.Error {
	return errs.Internal("Failed to hash password.").
		Reason("PASSWORD_HASH_FAILED").
		Wrap(err)
}

var (
	ErrUserDisabled = errs.PermissionDenied("User account is disabled.").
			Reason("USER_DISABLED")
	ErrUserScheduledDeletion = errs.FailedPrecondition("User account is scheduled for deletion.").
					Reason("USER_SCHEDULED_FOR_DELETION")
)
