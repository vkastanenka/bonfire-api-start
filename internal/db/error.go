package db

import (
	"errors"
	"fmt"
	"strings"

	"bonfire-api/internal/errs"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Entity string

const (
	EntityChannel         Entity = "channel"
	EntityChannelMember   Entity = "channel_member"
	EntityMessage         Entity = "message"
	EntityMessageReaction Entity = "message_reaction"
	EntityOutboxEvents    Entity = "outbox_event"
	EntityRelationship    Entity = "relationship"
	EntitySession         Entity = "session"
	EntityUser            Entity = "user"
)

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

func handleDbError(err error, entity Entity) error {
	if IsNotFoundError(err) {
		msg := fmt.Sprintf("The requested %s could not be found.", entity)
		return attachContext(errs.NotFound(msg), entity).Wrap(err)
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return handlePgError(err, pgErr, entity)
	}

	msg := fmt.Sprintf("An error occurred operating on %s.", entity)
	return attachContext(errs.Internal(msg), entity).Wrap(err)
}

func handlePgError(origErr error, pgErr *pgconn.PgError, entity Entity) error {
	switch pgErr.Code {
	case pgCodeUniqueViolation:
		return handleConstraint(origErr, pgErr, entity, errs.CodeAlreadyExists,
			"This %s is already taken.",
			fmt.Sprintf("A record for %s with those details already exists.", entity))

	case pgCodeNotNullViolation:
		return handleConstraint(origErr, pgErr, entity, errs.CodeInvalidArgument,
			"This field is required.",
			fmt.Sprintf("A required field is missing for %s.", entity))

	case pgCodeForeignKeyViolation:
		msg := fmt.Sprintf("Referenced target for %s does not exist or was deleted.", entity)
		return attachContext(errs.InvalidArgument(msg), entity).Wrap(origErr)

	case pgCodeCheckViolation:
		return handleConstraint(origErr, pgErr, entity, errs.CodeInvalidArgument,
			"Invalid value.",
			fmt.Sprintf("An operation on %s was rejected due to a constraint violation.", entity))

	case pgCodeStringDataTruncated:
		return handleConstraint(origErr, pgErr, entity, errs.CodeInvalidArgument,
			"Exceeds maximum allowed length.",
			fmt.Sprintf("A provided field for %s exceeds maximum length.", entity))

	case pgCodeNumericOutOfRange:
		msg := fmt.Sprintf("A numeric value for %s was out of range.", entity)
		return attachContext(errs.OutOfRange(msg), entity).Wrap(origErr)

	case pgCodeInvalidTextRepr:
		msg := fmt.Sprintf("Invalid data format provided for %s.", entity)
		return attachContext(errs.InvalidArgument(msg), entity).Wrap(origErr)

	case pgCodeSerializationFail, pgCodeDeadlockDetected:
		msg := fmt.Sprintf("Concurrent conflict while operating on %s. Please retry.", entity)
		return attachContext(errs.Aborted(msg), entity).Wrap(origErr)

	case pgCodeQueryCanceled:
		msg := fmt.Sprintf("Database operation on %s timed out.", entity)
		return attachContext(errs.DeadlineExceeded(msg), entity).Wrap(origErr)

	default:
		msg := fmt.Sprintf("An internal database error occurred while processing %s.", entity)
		return attachContext(errs.Internal(msg), entity).Wrap(origErr)
	}
}

func handleConstraint(
	origErr error,
	pgErr *pgconn.PgError,
	entity Entity,
	code errs.Code,
	fieldMsgTemplate string,
	fallbackMsg string,
) error {
	field, ok := getFieldName(pgErr, entity)
	if ok {
		formattedMsg := formatMessage(fieldMsgTemplate, field)
		return attachContext(errs.New(code, formattedMsg), entity).
			FieldViolation(field, formattedMsg, pgErr.Code).
			Wrap(origErr)
	}

	return attachContext(errs.New(code, fallbackMsg), entity).Wrap(origErr)
}

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

func sanitizeFieldName(raw string, entity Entity) string {
	suffixes := []string{"_key", "_fkey", "_check", "_pkey", "_idx", "_seq", "_unique"}
	for _, suffix := range suffixes {
		if strings.HasSuffix(raw, suffix) {
			raw = strings.TrimSuffix(raw, suffix)
			break
		}
	}

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

	return raw
}

func formatMessage(template string, field string) string {
	if strings.Contains(template, "%s") {
		return fmt.Sprintf(template, humanize(field))
	}
	return template
}

func humanize(s string) string {
	return strings.TrimSpace(strings.ReplaceAll(s, "_", " "))
}

func attachContext(e *errs.Error, entity Entity) *errs.Error {
	return e.Meta("entity", entity.String()).Resource("db", entity.String(), "", "")
}
