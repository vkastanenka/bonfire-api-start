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

	var appErr *apperr.Error
	if errors.As(err, &appErr) {
		return err
	}

	meta := apperr.WithMeta("entity", entity.String())
	resourceInfo := apperr.WithResourceInfo("db", entity.String(), "", "")
	options := apperr.WithOptions(meta, resourceInfo)

	if IsNotFoundError(err) {
		return apperr.NewNotFound(
			err,
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
			"This %s is already taken.",
			fmt.Sprintf("A record for %s with those details already exists.", entity.String()),
		)

	case pgCodeNotNullViolation:
		return p.handleConstraint(
			apperr.CodeInvalidArgument,
			"This field is required.",
			fmt.Sprintf("A required field is missing for %s.", entity.String()),
		)

	case pgCodeForeignKeyViolation:
		// Foreign keys mean a referenced record is missing, return Bad Request / Not Found
		return apperr.NewInvalidArgument(
			origErr,
			apperr.WithMessage(fmt.Sprintf("Referenced target for %s does not exist or was deleted.", entity.String())),
			options,
		)

	case pgCodeCheckViolation:
		return p.handleConstraint(
			apperr.CodeInvalidArgument,
			"Invalid value.",
			fmt.Sprintf("An operation on %s was rejected due to a constraint violation.", entity.String()),
		)

	case pgCodeStringDataTruncated:
		return p.handleConstraint(
			apperr.CodeInvalidArgument,
			"Exceeds maximum allowed length.",
			fmt.Sprintf("A provided field for %s exceeds maximum length.", entity.String()),
		)

	case pgCodeNumericOutOfRange:
		return apperr.NewOutOfRange(
			origErr,
			apperr.WithMessage(fmt.Sprintf("A numeric value for %s was out of range.", entity.String())),
			options,
		)

	case pgCodeInvalidTextRepr:
		return apperr.NewInvalidArgument(
			origErr,
			apperr.WithMessage(fmt.Sprintf("Invalid data format provided for %s.", entity.String())),
			options,
		)

	case pgCodeSerializationFail, pgCodeDeadlockDetected:
		return apperr.NewAborted(
			origErr,
			apperr.WithMessage(fmt.Sprintf("Concurrent conflict while operating on %s. Please retry.", entity.String())),
			options,
		)

	case pgCodeQueryCanceled:
		return apperr.NewDeadlineExceeded(
			origErr,
			apperr.WithMessage(fmt.Sprintf("Database operation on %s timed out.", entity.String())),
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
		return apperr.New(
			code,
			apperr.WithError(p.pgErr),
			apperr.WithFieldViolation(field, formattedMsg, ""),
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
	// 1. Strip known suffixes first
	suffixes := []string{"_key", "_fkey", "_check", "_pkey", "_idx", "_seq", "_unique"}
	for _, suffix := range suffixes {
		if strings.HasSuffix(raw, suffix) {
			raw = strings.TrimSuffix(raw, suffix)
			break
		}
	}

	// 2. Strip table/entity prefixes safely
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
