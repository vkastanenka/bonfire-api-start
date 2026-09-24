package user

import "bonfire-api/internal/pkg/errs"

func ErrInvalidPassword() *errs.Error {
	return errs.Unauthenticated("Invalid password.").
		Reason("INVALID_PASSWORD").
		FieldViolation("password", "Invalid password.", "INVALID_PASSWORD")
}

func ErrPasswordMismatch() *errs.Error {
	return errs.InvalidArgument("Passwords must match.").
		Reason("PASSWORD_MISMATCH").
		FieldViolation("password", "Passwords do not match.", "PASSWORD_MISMATCH")
}

func ErrPasswordHashFailed() *errs.Error {
	return errs.Internal("Failed to hash password.").
		Reason("PASSWORD_HASH_FAILED")
}

func ErrUserDisabled() *errs.Error {
	return errs.PermissionDenied("User account is disabled.").
		Reason("USER_DISABLED")
}

func ErrUserScheduledDeletion() *errs.Error {
	return errs.FailedPrecondition("User account is scheduled for deletion.").
		Reason("USER_SCHEDULED_FOR_DELETION")
}
