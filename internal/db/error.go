package db

import (
	"context"
	"errors"
	"net"
	"regexp"

	"bonfire-api/internal/pkg/errs"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Resource identifies the database resource associated with an operation or error.
type Resource string

const (
	ResourceChannel         Resource = "channel"
	ResourceChannelMember   Resource = "channel_member"
	ResourceMessage         Resource = "message"
	ResourceMessageReaction Resource = "message_reaction"
	ResourceOutboxEvent     Resource = "outbox_event"
	ResourceRelation        Resource = "relation"
	ResourceSession         Resource = "session"
	ResourceUser            Resource = "user"
)

func (e Resource) String() string { return string(e) }

const (
	pgCodeUniqueViolation     = "23505"
	pgCodeNotNullViolation    = "23502"
	pgCodeForeignKeyViolation = "23503"
	pgCodeCheckViolation      = "23514"
	pgCodeStringDataTruncated = "22001"
	pgCodeNumericOutOfRange   = "22003"
	pgCodeInvalidTextRepr     = "22P02"
	pgCodeSerializationFail   = "40001"
	pgCodeDeadlockDetected    = "40P01"
	pgCodeQueryCanceled       = "57014"
)

// Database error message and reason constants aligned with validator conventions
const (
	errDBAlreadyExists      = "This value is already taken."
	errDBRequired           = "This field is required."
	errDBForeignKeyNotFound = "The selected reference does not exist or was removed."
	errDBStringTooLong      = "Cannot exceed the maximum allowed length."
	errDBNumericOutOfRange  = "Must be within the allowed range."
	errDBInvalidData        = "Contains an invalid format or value."

	reasonDBAlreadyExists     = "ALREADY_EXISTS"
	reasonDBRequired          = "REQUIRED"
	reasonDBReferenceNotFound = "REFERENCE_NOT_FOUND"
	reasonDBStringTooLong     = "STRING_TOO_LONG"
	reasonDBOutofRange        = "OUT_OF_RANGE"
	reasonDBInvalidFormat     = "INVALID_FORMAT"
)

// Pre-compile the regex to extract column names dynamically from Postgres unique violation details
var uniqueKeyRegex = regexp.MustCompile(`Key \(([^)]+)\)=`)

// IsNotFoundError checks if the error represents a database record miss.
func IsNotFoundError(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

// IsAlreadyExistsError checks if the error represents a unique constraint violation.
func IsAlreadyExistsError(err error) bool {
	if err == nil {
		return false
	}

	if appErr := errs.As(err); appErr != nil {
		return appErr.Code == errs.CodeAlreadyExists
	}

	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == pgCodeUniqueViolation
}

// NewError transforms raw database errors into structured domain errors.
func NewError(err error, resource Resource) error {
	if err == nil {
		return nil
	}

	if appErr := errs.As(err); appErr != nil {
		return err
	}

	return handleDBError(err, resource)
}

// handleDBError classifies general database failures into domain errors, prioritizing record misses and Postgres-specific errors.
func handleDBError(err error, resource Resource) error {
	if IsNotFoundError(err) {
		return ErrWithResource(ErrDBNotFound(), resource).Wrap(err)
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return handlePGError(err, pgErr, resource)
	}

	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return ErrWithResource(ErrDBTimeout(), resource).Wrap(err)

	case errors.Is(err, context.Canceled):
		return ErrWithResource(ErrDBCanceled(), resource).Wrap(err)

	case isNetworkError(err):
		return ErrWithResource(ErrDBUnavailable(), resource).Wrap(err)

	default:
		return ErrWithResource(ErrDBInternal(), resource).Wrap(err)
	}
}

// handlePGError maps specific PostgreSQL error codes to structured domain error types.
func handlePGError(
	origErr error,
	pgErr *pgconn.PgError,
	resource Resource,
) error {
	if pgErr == nil {
		return ErrWithResource(ErrDBInternal(), resource).Wrap(origErr)
	}

	var e *errs.Error

	switch pgErr.Code {
	case pgCodeUniqueViolation:
		fieldName := resolveUniqueField(pgErr)
		e = ErrDBAlreadyExists().
			FieldViolation(
				fieldName,
				errDBAlreadyExists,
				reasonDBAlreadyExists,
			)

	case pgCodeNotNullViolation:
		e = ErrDBRequiredField().
			FieldViolation(
				pgErr.ColumnName,
				errDBRequired,
				reasonDBRequired,
			)

	case pgCodeForeignKeyViolation:
		e = ErrDBForeignKey().
			FieldViolation(
				pgErr.ColumnName,
				errDBForeignKeyNotFound,
				reasonDBReferenceNotFound,
			)

	case pgCodeCheckViolation:
		e = ErrDBConstraint()

	case pgCodeStringDataTruncated:
		e = ErrDBStringTooLong().
			FieldViolation(
				pgErr.ColumnName,
				errDBStringTooLong,
				reasonDBStringTooLong,
			)

	case pgCodeNumericOutOfRange:
		e = ErrDBNumericOutOfRange().
			FieldViolation(
				pgErr.ColumnName,
				errDBNumericOutOfRange,
				reasonDBOutofRange,
			)

	case pgCodeInvalidTextRepr:
		e = ErrDBInvalidData().
			FieldViolation(
				pgErr.ColumnName,
				errDBInvalidData,
				reasonDBInvalidFormat,
			)

	case pgCodeSerializationFail,
		pgCodeDeadlockDetected:
		e = ErrDBConcurrency()

	case pgCodeQueryCanceled:
		e = ErrDBTimeout()

	default:
		e = ErrDBInternal()
	}

	e = ErrWithResource(e, resource)

	return e.Wrap(origErr)
}

// resolveUniqueField extracts the conflicting column name dynamically without hardcoded maps.
func resolveUniqueField(pgErr *pgconn.PgError) string {
	if pgErr.ColumnName != "" {
		return pgErr.ColumnName
	}

	if matches := uniqueKeyRegex.FindStringSubmatch(pgErr.Detail); len(matches) > 1 {
		return matches[1]
	}

	return pgErr.ConstraintName
}

func ErrWithResource(e *errs.Error, resource Resource) *errs.Error {
	return e.Resource("db", resource.String(), "", "")
}

func isNetworkError(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr)
}
