package repository

import (
	"bonfire-api/internal/apperr"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Scope string

const (
	ScopeSession     Scope = "session"
	ScopeUser        Scope = "user"
	ScopeUserProfile Scope = "user_profile"
)

func (r Scope) String() string {
	return string(r)
}

var ErrNotFound = errors.New("store: resource not found")

func IsNotFoundError(err error) bool {
	return errors.Is(err, pgx.ErrNoRows) || errors.Is(err, ErrNotFound)
}

func NewError(err error, scope Scope) error {
	if err == nil {
		return nil
	}

	var appErr *apperr.Error
	if errors.As(err, &appErr) {
		return err
	}

	if IsNotFoundError(err) {
		return apperr.NewNotFound(err, fmt.Sprintf("%s was not found in store.", scope))
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			return handleUniqueViolation(pgErr, scope)

		case "23502": // not_null_violation
			return handleNotNullViolation(pgErr, scope)

		case "23503": // foreign_key_violation
			return handleForeignKeyViolation(pgErr, scope)

		case "23514": // check_violation
			return handleCheckViolation(pgErr, scope)

		case "22001": // string_data_right_truncation
			return handleLengthTruncation(pgErr, scope)

		case "22003": // numeric_value_out_of_range
			return apperr.NewInvalidInput(err, "A provided number is out of the acceptable range.")

		case "22P02": // invalid_text_representation (e.g., bad UUIDs)
			return apperr.NewInvalidInput(err, "The data format is invalid or malformed.")

		case "40001", "40P01": // serialization_failure & deadlock_detected
			return apperr.NewConflict(err, "A resource conflict occurred. Please retry your request.")

		case "57014": // query_canceled
			return apperr.NewRequestTimeout(err, "The database operation timed out or was canceled.")
		}
	}

	return apperr.NewInternal(err, "")
}

func handleUniqueViolation(pgErr *pgconn.PgError, scope Scope) error {
	if pgErr.ConstraintName != "" && strings.HasSuffix(pgErr.ConstraintName, "_key") {
		field := cleanIdentifier(pgErr.ConstraintName, "_key", scope)
		reason := fmt.Sprintf("This %s is already taken.", humanize(field))

		return apperr.NewInvalidInput(pgErr, "", apperr.Param(field, reason))
	}

	return apperr.NewConflict(pgErr, fmt.Sprintf("A conflict occurred. This %s already exists.", scope))
}

func handleNotNullViolation(pgErr *pgconn.PgError, scope Scope) error {
	if pgErr.ColumnName != "" {
		field := cleanIdentifier(pgErr.ColumnName, "", scope)
		return apperr.NewInvalidInput(pgErr, "", apperr.Param(field, "This field is required."))
	}

	return apperr.NewInvalidInput(pgErr, "")
}

func handleForeignKeyViolation(pgErr *pgconn.PgError, scope Scope) error {
	if pgErr.ConstraintName != "" && strings.HasSuffix(pgErr.ConstraintName, "_fkey") {
		field := cleanIdentifier(pgErr.ConstraintName, "_fkey", scope)
		reason := fmt.Sprintf("Must reference a valid %s.", humanize(field))

		return apperr.NewInvalidInput(pgErr, "", apperr.Param(field, reason))
	}

	return apperr.NewInvalidInput(pgErr, fmt.Sprintf("A referenced %s record does not exist.", scope))
}

func handleCheckViolation(pgErr *pgconn.PgError, scope Scope) error {
	if pgErr.ConstraintName != "" && strings.HasSuffix(pgErr.ConstraintName, "_check") {
		field := cleanIdentifier(pgErr.ConstraintName, "_check", scope)
		reason := fmt.Sprintf("Invalid value for constraint: %s", humanize(field))

		return apperr.NewInvalidInput(pgErr, "", apperr.Param(field, reason))
	}

	return apperr.NewInvalidInput(pgErr, "")
}

func handleLengthTruncation(pgErr *pgconn.PgError, scope Scope) error {
	if pgErr.ColumnName != "" {
		field := cleanIdentifier(pgErr.ColumnName, "", scope)
		return apperr.NewInvalidInput(pgErr, "", apperr.Param(field, "Cannot be longer than the maximum allowed length."))
	}

	return apperr.NewInvalidInput(pgErr, "A provided text field exceeds the maximum allowed length.")
}

func cleanIdentifier(identifier string, suffix string, scope Scope) string {
	if suffix != "" {
		identifier = strings.TrimSuffix(identifier, suffix)
	}

	prefix := scope.String() + "_"
	pluralPrefix := scope.String() + "s_"

	if strings.HasPrefix(identifier, pluralPrefix) {
		identifier = strings.TrimPrefix(identifier, pluralPrefix)
	} else if strings.HasPrefix(identifier, prefix) {
		identifier = strings.TrimPrefix(identifier, prefix)
	}

	return identifier
}

func humanize(s string) string {
	return strings.ReplaceAll(s, "_", " ")
}
