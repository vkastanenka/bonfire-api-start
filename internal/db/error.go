package db

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"

	"bonfire-api/internal/pkg/errs"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Entity string

const (
	EntityChannel         Entity = "channel"
	EntityChannelMember   Entity = "channel_member"
	EntityMessage         Entity = "message"
	EntityMessageReaction Entity = "message_reaction"
	EntityOutboxEvent     Entity = "outbox_event"
	EntityRelation        Entity = "relation"
	EntitySession         Entity = "session"
	EntityUser            Entity = "user"
)

// String returns the string representation of the database Entity.
func (e Entity) String() string { return string(e) }

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

// ErrNotFound represents a standard database record miss and wraps pgx.ErrNoRows for direct comparison.
var ErrNotFound = fmt.Errorf("db: record not found: %w", pgx.ErrNoRows)

// IsNotFoundError checks if the error represents a database record miss.
func IsNotFoundError(err error) bool {
	return errors.Is(err, pgx.ErrNoRows) || errors.Is(err, ErrNotFound)
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

// NewError transforms raw database errors into structured domain errors with entity metadata.
func NewError(err error, entity Entity) error {
	if err == nil {
		return nil
	}

	if appErr := errs.As(err); appErr != nil {
		return err
	}

	return handleDbError(err, entity)
}

// handleDbError classifies general database failures into domain errors, prioritizing record misses and Postgres-specific errors.
func handleDbError(err error, entity Entity) error {
	if IsNotFoundError(err) {
		return attachContext(errs.NotFound("The requested resource could not be found."), entity, "record missing or deleted").
			Reason("RESOURCE_NOT_FOUND").
			Wrap(err)
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return handlePgError(err, pgErr, entity)
	}

	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return attachContext(errs.DeadlineExceeded("Database operation timed out."), entity, "query execution deadline exceeded").
			Reason("DB_TIMEOUT").
			Wrap(err)

	case errors.Is(err, context.Canceled):
		return attachContext(errs.Aborted("Database operation was canceled."), entity, "operation canceled by client").
			Reason("DB_CANCELED").
			Wrap(err)

	case isNetworkError(err):
		return attachContext(errs.Unavailable("Database service is temporarily unavailable."), entity, "connection failed").
			Reason("DB_UNAVAILABLE").
			Wrap(err)

	default:
		return attachContext(errs.Internal("An internal database error occurred."), entity, "unhandled database failure").
			Reason("DB_INTERNAL_ERROR").
			Wrap(err)
	}
}

// handlePgError maps specific PostgreSQL error codes to structured domain error types.
func handlePgError(origErr error, pgErr *pgconn.PgError, entity Entity) error {
	switch pgErr.Code {
	case pgCodeUniqueViolation:
		return handleConstraint(origErr, pgErr, entity, errs.CodeAlreadyExists,
			"ALREADY_EXISTS",
			"A resource with these details already exists.")

	case pgCodeNotNullViolation:
		return handleConstraint(origErr, pgErr, entity, errs.CodeInvalidArgument,
			"REQUIRED_FIELD_MISSING",
			"A required field is missing.")

	case pgCodeForeignKeyViolation:
		e := attachContext(errs.InvalidArgument("Referenced entity does not exist or was deleted."), entity, "foreign key target missing").
			Reason("FOREIGN_KEY_VIOLATION")
		if pgErr.ConstraintName != "" {
			e = e.Meta("constraint", pgErr.ConstraintName)
		}
		return e.Wrap(origErr)

	case pgCodeCheckViolation:
		return handleConstraint(origErr, pgErr, entity, errs.CodeInvalidArgument,
			"CONSTRAINT_VIOLATION",
			"The provided value violates a domain constraint.")

	case pgCodeStringDataTruncated:
		return handleConstraint(origErr, pgErr, entity, errs.CodeInvalidArgument,
			"STRING_TOO_LONG",
			"A provided field exceeds the maximum allowed length.")

	case pgCodeNumericOutOfRange:
		return attachContext(errs.OutOfRange("A numeric value was out of allowed range."), entity, "numeric field overflow").
			Reason("NUMERIC_OUT_OF_RANGE").
			Wrap(origErr)

	case pgCodeInvalidTextRepr:
		return attachContext(errs.InvalidArgument("Invalid data format provided."), entity, "malformed column value").
			Reason("INVALID_DATA_FORMAT").
			Wrap(origErr)

	case pgCodeSerializationFail, pgCodeDeadlockDetected:
		return attachContext(errs.Aborted("Concurrent modification conflict. Please retry."), entity, "serialization failure or deadlock").
			Reason("CONCURRENCY_CONFLICT").
			Wrap(origErr)

	case pgCodeQueryCanceled:
		return attachContext(errs.DeadlineExceeded("Database operation timed out."), entity, "statement execution canceled").
			Reason("QUERY_CANCELED").
			Wrap(origErr)

	default:
		return attachContext(errs.Internal("An internal database error occurred."), entity, "postgres engine error").
			Reason("DB_INTERNAL_ERROR").
			Wrap(origErr)
	}
}

// handleConstraint constructs a domain error for database constraint violations, embedding field and table metadata.
func handleConstraint(
	origErr error,
	pgErr *pgconn.PgError,
	entity Entity,
	code errs.Code,
	reason string,
	staticMsg string,
) error {
	e := attachContext(errs.New(code, staticMsg), entity, "constraint violation").
		Reason(reason)

	if pgErr.ConstraintName != "" {
		e = e.Meta("constraint", pgErr.ConstraintName)
	}
	if pgErr.TableName != "" {
		e = e.Meta("table", pgErr.TableName)
	}

	field, ok := getFieldName(pgErr, entity)
	if ok {
		e = e.Meta("field", field).
			FieldViolation(field, staticMsg, pgErr.Code)
	}

	return e.Wrap(origErr)
}

// getFieldName extracts and cleans the target column or constraint field name from a Postgres error.
func getFieldName(pgErr *pgconn.PgError, entity Entity) (string, bool) {
	if pgErr == nil {
		return "", false
	}

	raw := pgErr.ConstraintName
	if raw == "" {
		raw = pgErr.ColumnName
	}
	if raw == "" {
		return "", false
	}

	return sanitizeFieldName(raw, entity), true
}

// sanitizeFieldName strips entity prefixes and database suffixes to isolate the raw property name.
func sanitizeFieldName(raw string, entity Entity) string {
	e := entity.String()
	if e != "" {
		prefixes := []string{
			"fk_" + e + "_",
			"fk_" + e + "s_",
			e + "_",
			e + "s_",
		}
		for _, prefix := range prefixes {
			if strings.HasPrefix(raw, prefix) {
				raw = strings.TrimPrefix(raw, prefix)
				break
			}
		}
	}

	suffixes := []string{"_key", "_fkey", "_check", "_pkey", "_idx", "_seq", "_unique"}
	for _, suffix := range suffixes {
		if strings.HasSuffix(raw, suffix) {
			raw = strings.TrimSuffix(raw, suffix)
			break
		}
	}

	return raw
}

// isNetworkError determines whether an error stems from underlying network or I/O failure.
func isNetworkError(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr)
}

// attachContext enriches a domain error with entity metadata and standardized database resource context.
func attachContext(e *errs.Error, entity Entity, desc string) *errs.Error {
	return e.Meta("entity", entity.String()).Resource("db", entity.String(), "", desc)
}
