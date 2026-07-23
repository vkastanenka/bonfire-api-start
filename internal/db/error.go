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
	EntityChannel       Entity = "channel"
	EntityChannelMember Entity = "channel_member"
	EntityOutboxEvent   Entity = "outbox_event"
	EntityRelationship  Entity = "relationship"
	EntitySession       Entity = "session"
	EntityUser          Entity = "user"
	EntityUserProfile   Entity = "user_profile"
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

func NewError(err error, entity Entity) error {
	if err == nil {
		return nil
	}

	if appErr := errs.As(err); appErr != nil {
		return err
	}

	if IsNotFoundError(err) {
		return attachContext(
			errs.NotFound(fmt.Sprintf("The requested %s could not be found.", entity.String())),
			entity,
		).Wrap(err)
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return handlePgError(err, pgErr, entity)
	}

	return attachContext(
		errs.Internal(fmt.Sprintf("An error occurred operating on %s.", entity.String())),
		entity,
	).Wrap(err)
}

func IsNotFoundError(err error) bool { return errors.Is(err, pgx.ErrNoRows) }

func attachContext(e *errs.Error, entity Entity) *errs.Error {
	return e.Meta("entity", entity.String()).Resource("db", entity.String(), "", "")
}

func handlePgError(origErr error, pgErr *pgconn.PgError, entity Entity) error {
	p := dbErrorParams{
		origErr: origErr,
		pgErr:   pgErr,
		entity:  entity,
	}

	switch pgErr.Code {
	case pgCodeUniqueViolation:
		return p.handleConstraint(
			errs.CodeAlreadyExists,
			"This %s is already taken.",
			fmt.Sprintf("A record for %s with those details already exists.", entity.String()),
		)

	case pgCodeNotNullViolation:
		return p.handleConstraint(
			errs.CodeInvalidArgument,
			"This field is required.",
			fmt.Sprintf("A required field is missing for %s.", entity.String()),
		)

	case pgCodeForeignKeyViolation:
		return attachContext(
			errs.InvalidArgument(fmt.Sprintf("Referenced target for %s does not exist or was deleted.", entity.String())),
			entity,
		).Wrap(origErr)

	case pgCodeCheckViolation:
		return p.handleConstraint(
			errs.CodeInvalidArgument,
			"Invalid value.",
			fmt.Sprintf("An operation on %s was rejected due to a constraint violation.", entity.String()),
		)

	case pgCodeStringDataTruncated:
		return p.handleConstraint(
			errs.CodeInvalidArgument,
			"Exceeds maximum allowed length.",
			fmt.Sprintf("A provided field for %s exceeds maximum length.", entity.String()),
		)

	case pgCodeNumericOutOfRange:
		return attachContext(
			errs.OutOfRange(fmt.Sprintf("A numeric value for %s was out of range.", entity.String())),
			entity,
		).Wrap(origErr)

	case pgCodeInvalidTextRepr:
		return attachContext(
			errs.InvalidArgument(fmt.Sprintf("Invalid data format provided for %s.", entity.String())),
			entity,
		).Wrap(origErr)

	case pgCodeSerializationFail, pgCodeDeadlockDetected:
		return attachContext(
			errs.Aborted(fmt.Sprintf("Concurrent conflict while operating on %s. Please retry.", entity.String())),
			entity,
		).Wrap(origErr)

	case pgCodeQueryCanceled:
		return attachContext(
			errs.DeadlineExceeded(fmt.Sprintf("Database operation on %s timed out.", entity.String())),
			entity,
		).Wrap(origErr)

	default:
		return attachContext(
			errs.Internal(fmt.Sprintf("An internal database error occurred while processing %s.", entity.String())),
			entity,
		).Wrap(origErr)
	}
}

type dbErrorParams struct {
	origErr error
	pgErr   *pgconn.PgError
	entity  Entity
}

func (p dbErrorParams) handleConstraint(
	code errs.Code,
	fieldMsgTemplate string,
	fallbackMsg string,
) error {
	field, ok := getFieldName(p.pgErr, p.entity)
	if ok {
		formattedMsg := formatMessage(fieldMsgTemplate, field)
		return attachContext(errs.New(code, formattedMsg), p.entity).
			FieldViolation(field, formattedMsg, p.pgErr.Code).
			Wrap(p.origErr)
	}

	return attachContext(errs.New(code, fallbackMsg), p.entity).Wrap(p.origErr)
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
