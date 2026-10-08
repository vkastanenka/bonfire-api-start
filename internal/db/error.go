package db

import (
	"context"
	"errors"
	"net"

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
		return withDBResource(ErrDBNotFound(), resource).Wrap(err)
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return handlePGError(err, pgErr, resource)
	}

	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return withDBResource(ErrDBTimeout(), resource).Wrap(err)

	case errors.Is(err, context.Canceled):
		return withDBResource(ErrDBCanceled(), resource).Wrap(err)

	case isNetworkError(err):
		return withDBResource(ErrDBUnavailable(), resource).Wrap(err)

	default:
		return withDBResource(ErrDBInternal(), resource).Wrap(err)
	}
}

// handlePGError maps specific PostgreSQL error codes to structured domain error types.
func handlePGError(
	origErr error,
	pgErr *pgconn.PgError,
	resource Resource,
) error {
	if pgErr == nil {
		return withDBResource(ErrDBInternal(), resource).Wrap(origErr)
	}

	var e *errs.Error

	switch pgErr.Code {
	case pgCodeUniqueViolation:
		e = ErrDBAlreadyExists().
			FieldViolation(
				pgErr.ColumnName,
				"Value is already in use.",
				"ALREADY_EXISTS",
			)

	case pgCodeNotNullViolation:
		e = ErrDBRequiredField().
			FieldViolation(
				pgErr.ColumnName,
				"Field is required.",
				"REQUIRED",
			)

	case pgCodeForeignKeyViolation:
		e = ErrDBForeignKey().
			FieldViolation(
				pgErr.ColumnName,
				"Referenced resource does not exist or was deleted.",
				"REFERENCE_NOT_FOUND",
			)

	case pgCodeCheckViolation:
		e = ErrDBConstraint()

	case pgCodeStringDataTruncated:
		e = ErrDBStringTooLong().
			FieldViolation(
				pgErr.ColumnName,
				"Field exceeds the maximum allowed length.",
				"STRING_TOO_LONG",
			)

	case pgCodeNumericOutOfRange:
		e = ErrDBNumericOutOfRange().
			FieldViolation(
				pgErr.ColumnName,
				"Value is outside the allowed range.",
				"OUT_OF_RANGE",
			)

	case pgCodeInvalidTextRepr:
		e = ErrDBInvalidData().
			FieldViolation(
				pgErr.ColumnName,
				"Field contains an invalid value.",
				"INVALID_FORMAT",
			)

	case pgCodeSerializationFail,
		pgCodeDeadlockDetected:
		e = ErrDBConcurrency()

	case pgCodeQueryCanceled:
		e = ErrDBTimeout()

	default:
		e = ErrDBInternal()
	}

	e = withDBResource(e, resource)

	return e.Wrap(origErr)
}

func withDBResource(e *errs.Error, resource Resource) *errs.Error {
	return e.Resource("db", resource.String(), "", "")
}

func isNetworkError(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr)
}
