package db

import "bonfire-api/internal/pkg/errs"

const (
	ErrReasonDBConfigInvalid     = "DB_CONFIG_INVALID"
	ErrReasonDBURLInvalid        = "DB_URL_INVALID"
	ErrReasonDBPoolCreateFailed  = "DB_POOL_CREATE_FAILED"
	ErrReasonDBPingFailed        = "DB_PING_FAILED"
	ErrReasonDBTXStartFailed     = "DB_TX_START_FAILED"
	ErrReasonDBTXCommitFailed    = "DB_TX_COMMIT_FAILED"
	ErrReasonDBNotFound          = "DB_NOT_FOUND"
	ErrReasonDBAlreadyExists     = "DB_ALREADY_EXISTS"
	ErrReasonDBRequiredField     = "DB_REQUIRED_FIELD_MISSING"
	ErrReasonDBForeignKey        = "DB_FOREIGN_KEY_VIOLATION"
	ErrReasonDBConstraint        = "DB_CONSTRAINT_VIOLATION"
	ErrReasonDBStringTooLong     = "DB_STRING_TOO_LONG"
	ErrReasonDBNumericOutOfRange = "DB_NUMERIC_OUT_OF_RANGE"
	ErrReasonDBInvalidData       = "DB_INVALID_DATA_FORMAT"
	ErrReasonDBConcurrency       = "DB_CONCURRENCY_CONFLICT"
	ErrReasonDBTimeout           = "DB_TIMEOUT"
	ErrReasonDBCanceled          = "DB_CANCELED"
	ErrReasonDBUnavailable       = "DB_UNAVAILABLE"
	ErrReasonDBInternal          = "DB_INTERNAL_ERROR"
)

// ErrDBConfigInvalid returns an InvalidArgument error when the database connection string is empty.
func ErrDBConfigInvalid() *errs.Error {
	return errs.InvalidArgument("Database connection string cannot be empty.").
		Reason(ErrReasonDBConfigInvalid).
		FieldViolation(
			"conn_string",
			"Connection string is required.",
			"REQUIRED",
		)
}

// ErrDBURLInvalid returns an InvalidArgument error when parsing the database connection string fails.
func ErrDBURLInvalid() *errs.Error {
	return errs.InvalidArgument("Invalid database connection string.").
		Reason(ErrReasonDBURLInvalid).
		FieldViolation(
			"conn_string",
			"Failed to parse database connection string.",
			"INVALID_FORMAT",
		)
}

// ErrDBPoolCreateFailed returns an Unavailable error when the database connection pool cannot be created.
func ErrDBPoolCreateFailed() *errs.Error {
	return errs.Unavailable("Failed to create database pool.").
		Reason(ErrReasonDBPoolCreateFailed)
}

// ErrDBPingFailed returns an Unavailable error when the database connectivity check fails during setup.
func ErrDBPingFailed() *errs.Error {
	return errs.Unavailable("Database connection ping failed.").
		Reason(ErrReasonDBPingFailed)
}

// ErrDBTXStartFailed returns an Internal error when a database transaction cannot be started.
func ErrDBTXStartFailed() *errs.Error {
	return errs.Internal("Failed to begin database transaction.").
		Reason(ErrReasonDBTXStartFailed)
}

// ErrDBTXCommitFailed returns an Internal error when a database transaction cannot be committed.
func ErrDBTXCommitFailed() *errs.Error {
	return errs.Internal("Failed to commit database transaction.").
		Reason(ErrReasonDBTXCommitFailed)
}

// ErrDBNotFound returns a NotFound error when a requested database record does not exist.
func ErrDBNotFound() *errs.Error {
	return errs.NotFound("The requested resource could not be found.").
		Reason(ErrReasonDBNotFound)
}

// ErrDBAlreadyExists returns an AlreadyExists error when a database uniqueness constraint is violated.
func ErrDBAlreadyExists() *errs.Error {
	return errs.AlreadyExists("A resource with these details already exists.").
		Reason(ErrReasonDBAlreadyExists)
}

// ErrDBRequiredField returns an InvalidArgument error when a required database field is missing.
func ErrDBRequiredField() *errs.Error {
	return errs.InvalidArgument("A required field is missing.").
		Reason(ErrReasonDBRequiredField)
}

// ErrDBForeignKey returns an InvalidArgument error when a referenced database resource does not exist.
func ErrDBForeignKey() *errs.Error {
	return errs.InvalidArgument("Referenced resource does not exist or was deleted.").
		Reason(ErrReasonDBForeignKey)
}

// ErrDBConstraint returns an InvalidArgument error when a database check constraint is violated.
func ErrDBConstraint() *errs.Error {
	return errs.InvalidArgument("The provided value violates a domain constraint.").
		Reason(ErrReasonDBConstraint)
}

// ErrDBStringTooLong returns an InvalidArgument error when a string exceeds its database column limit.
func ErrDBStringTooLong() *errs.Error {
	return errs.InvalidArgument("A provided field exceeds the maximum allowed length.").
		Reason(ErrReasonDBStringTooLong)
}

// ErrDBNumericOutOfRange returns an OutOfRange error when a numeric value exceeds its database type range.
func ErrDBNumericOutOfRange() *errs.Error {
	return errs.OutOfRange("A numeric value was out of allowed range.").
		Reason(ErrReasonDBNumericOutOfRange)
}

// ErrDBInvalidData returns an InvalidArgument error when a database value has an invalid format.
func ErrDBInvalidData() *errs.Error {
	return errs.InvalidArgument("Invalid data format provided.").
		Reason(ErrReasonDBInvalidData)
}

// ErrDBConcurrency returns an Aborted error when a database operation encounters a concurrency conflict.
func ErrDBConcurrency() *errs.Error {
	return errs.Aborted("Concurrent modification conflict. Please retry.").
		Reason(ErrReasonDBConcurrency)
}

// ErrDBTimeout returns a DeadlineExceeded error when a database operation exceeds its allowed time.
func ErrDBTimeout() *errs.Error {
	return errs.DeadlineExceeded("Database operation timed out.").
		Reason(ErrReasonDBTimeout)
}

// ErrDBCanceled returns a Canceled error when a database operation is canceled by its context.
func ErrDBCanceled() *errs.Error {
	return errs.Cancelled("Database operation was canceled.").
		Reason(ErrReasonDBCanceled)
}

// ErrDBUnavailable returns an Unavailable error when the database service cannot be reached or is temporarily unavailable.
func ErrDBUnavailable() *errs.Error {
	return errs.Unavailable("Database service is temporarily unavailable.").
		Reason(ErrReasonDBUnavailable)
}

// ErrDBInternal returns an Internal error for unexpected database failures that cannot be classified more specifically.
func ErrDBInternal() *errs.Error {
	return errs.Internal("An internal database error occurred.").
		Reason(ErrReasonDBInternal)
}
