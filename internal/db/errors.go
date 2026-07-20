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

	resourceInfo := apperr.WithResourceInfo("db", entity.String(), "", "")

	if IsNotFoundError(err) {
		return apperr.NewNotFound(err,
			"",
			"The requested {entity} could not be found.",
			apperr.WithParams(map[string]string{"entity": entity.String()}),
			resourceInfo,
		)
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return handlePgError(err, pgErr, entity, resourceInfo)
	}

	return apperr.NewInternal(err, "", "", resourceInfo)
}

func IsNotFoundError(err error) bool { return errors.Is(err, pgx.ErrNoRows) }

func handlePgError(origErr error, pgErr *pgconn.PgError, entity Entity, resourceInfo apperr.Option) error {
	p := dbErrorParams{
		origErr:      origErr,
		pgErr:        pgErr,
		entity:       entity,
		resourceInfo: resourceInfo,
	}

	switch pgErr.Code {
	case pgCodeUniqueViolation:
		return p.handleConstraint(
			apperr.CodeAlreadyExists,
			"This %s is already taken.",
			"A record for {entity} with those details already exists.",
		)

	case pgCodeNotNullViolation:
		return p.handleConstraint(
			apperr.CodeInvalidArgument,
			"This field is required.",
			"A required field is missing for {entity}.",
		)

	case pgCodeForeignKeyViolation:
		return p.handleConstraint(
			apperr.CodeInvalidArgument,
			"Must reference a valid %s.",
			"A referenced record required for {entity} does not exist.",
		)

	case pgCodeCheckViolation:
		return p.handleConstraint(
			apperr.CodeInvalidArgument,
			"Invalid value for constraint: %s.",
			"An operation on {entity} was rejected due to a constraint violation.",
		)

	case pgCodeStringDataTruncated:
		return p.handleConstraint(
			apperr.CodeInvalidArgument,
			"Cannot be longer than the maximum allowed length.",
			"A provided field for {entity} exceeds the maximum allowed length.",
		)

	case pgCodeNumericOutOfRange:
		return apperr.NewOutOfRange(
			origErr,
			"",
			"A numeric value for {entity} was out of the allowed range.",
			apperr.WithParams(map[string]string{"entity": entity.String()}),
			resourceInfo,
		)

	case pgCodeInvalidTextRepr:
		return apperr.NewInvalidArgument(
			origErr,
			"",
			"One or more provided parameters for {entity} are formatted incorrectly.",
			apperr.WithParams(map[string]string{"entity": entity.String()}),
			resourceInfo,
		)

	case pgCodeSerializationFail, pgCodeDeadlockDetected:
		return apperr.NewAborted(
			origErr,
			"",
			"The operation on {entity} could not be completed due to a concurrent conflict. Please try again.",
			apperr.WithParams(map[string]string{"entity": entity.String()}),
			resourceInfo,
		)

	case pgCodeQueryCanceled:
		return apperr.NewDeadlineExceeded(
			origErr,
			"",
			"The requested database operation on {entity} timed out.",
			apperr.WithParams(map[string]string{"entity": entity.String()}),
			resourceInfo,
		)

	default:
		return apperr.NewInternal(
			origErr,
			"",
			"An internal database error occurred while processing {entity}.",
			apperr.WithParams(map[string]string{"entity": entity.String()}),
			resourceInfo,
		)
	}
}

type dbErrorParams struct {
	origErr      error
	pgErr        *pgconn.PgError
	entity       Entity
	resourceInfo apperr.Option
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
			"",
			"Validation failed.",
			p.resourceInfo,
			fieldViolation,
			apperr.WithError(p.pgErr),
		)
	}

	return apperr.New(
		code,
		"",
		fallbackMsg,
		apperr.WithParams(map[string]string{"entity": p.entity.String()}),
		p.resourceInfo,
		apperr.WithError(p.pgErr),
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
