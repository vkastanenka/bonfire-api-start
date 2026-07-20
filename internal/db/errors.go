package db

import (
	"errors"
	"fmt"
	"strings"

	"bonfire-api/internal/apperr"

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

func (e Entity) String() string {
	return string(e)
}

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

	var appErr *apperr.Error
	if errors.As(err, &appErr) {
		return err
	}

	meta := apperr.WithMeta("entity", entity.String())
	resourceInfo := apperr.WithResourceInfo("db", entity.String(), "", "")
	options := apperr.WithOptions(meta, resourceInfo)

	if IsNotFoundError(err) {
		return apperr.NewNotFound(err,
			apperr.WithMessage(fmt.Sprintf("The requested %s could not be found.", entity.String())),
			options,
		)
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return handlePgError(err, pgErr, entity, options)
	}

	return apperr.NewInternal(err, options)
}

func IsNotFoundError(err error) bool { return errors.Is(err, pgx.ErrNoRows) }

func handlePgError(origErr error, pgErr *pgconn.PgError, entity Entity, options apperr.Option) error {
	p := dbErrorParams{
		origErr: origErr,
		pgErr:   pgErr,
		entity:  entity,
		options: options,
	}

	switch pgErr.Code {
	case pgCodeUniqueViolation:
		return p.handleConstraint(
			apperr.CodeAlreadyExists,
			fmt.Sprintf("This %s is already taken.", entity.String()),
			fmt.Sprintf("A record for %s with those details already exists.", entity.String()),
		)

	case pgCodeNotNullViolation:
		return p.handleConstraint(
			apperr.CodeInvalidArgument,
			"This field is required.",
			fmt.Sprintf("A required field is missing for %s.", entity.String()),
		)

	case pgCodeForeignKeyViolation:
		return p.handleConstraint(
			apperr.CodeInvalidArgument,
			fmt.Sprintf("Must reference a valid %s.", entity.String()),
			fmt.Sprintf("A referenced record required for %s does not exist.", entity.String()),
		)

	case pgCodeCheckViolation:
		return p.handleConstraint(
			apperr.CodeInvalidArgument,
			"Invalid value for constraint.",
			fmt.Sprintf("An operation on %s was rejected due to a constraint violation.", entity.String()),
		)

	case pgCodeStringDataTruncated:
		return p.handleConstraint(
			apperr.CodeInvalidArgument,
			"Cannot be longer than the maximum allowed length.",
			fmt.Sprintf("A provided field for %s exceeds the maximum allowed length.", entity.String()),
		)

	case pgCodeNumericOutOfRange:
		return apperr.NewOutOfRange(
			origErr,
			apperr.WithMessage(fmt.Sprintf("A numeric value for %s was out of the allowed range.", entity.String())),
			options,
		)

	case pgCodeInvalidTextRepr:
		return apperr.NewInvalidArgument(
			origErr,
			apperr.WithMessage(fmt.Sprintf("One or more provided parameters for %s are formatted incorrectly.", entity.String())),
			options,
		)

	case pgCodeSerializationFail, pgCodeDeadlockDetected:
		return apperr.NewAborted(
			origErr,
			apperr.WithMessage(fmt.Sprintf("The operation on %s could not be completed due to a concurrent conflict. Please try again.", entity.String())),
			options,
		)

	case pgCodeQueryCanceled:
		return apperr.NewDeadlineExceeded(
			origErr,
			apperr.WithMessage(fmt.Sprintf("The requested database operation on %s timed out.", entity.String())),
			options,
		)

	default:
		return apperr.NewInternal(
			origErr,
			apperr.WithMessage(fmt.Sprintf("An internal database error occurred while processing %s.", entity.String())),
			options,
		)
	}
}

type dbErrorParams struct {
	origErr error
	pgErr   *pgconn.PgError
	entity  Entity
	options apperr.Option
}

func (p dbErrorParams) handleConstraint(
	code apperr.Code,
	fieldMsgTemplate string,
	fallbackMsg string,
) error {
	field, ok := getFieldName(p.pgErr, p.entity)
	if ok {
		formattedMsg := formatMessage(fieldMsgTemplate, field)
		fieldViolation := apperr.WithFieldViolation(field, formattedMsg, "")

		return apperr.New(
			code,
			apperr.WithError(p.pgErr),
			fieldViolation,
			p.options,
		)
	}

	return apperr.New(
		code,
		apperr.WithError(p.pgErr),
		apperr.WithMessage(fallbackMsg),
		p.options,
	)
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

	entityName := entity.String()
	if entityName != "" {
		prefixes := []string{
			"fk_" + entityName + "_",
			"fk_" + entityName + "s_",
			entityName + "_",
			entityName + "s_",
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
