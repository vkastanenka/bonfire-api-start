package repository

import (
	"bonfire-api/internal/apperr"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Scope represents an explicit database domain resource.
type Scope string

const (
	ScopeProfile Scope = "profile"
	ScopeSession Scope = "session"
	ScopeUser    Scope = "user"
)

func (r Scope) String() string {
	return string(r)
}

// ErrNotFound is a package-level sentinel error for missing records.
// Defining this as a variable ensures errors.Is comparisons succeed.
var ErrNotFound = errors.New("resource not found")

// IsNotFoundError checks if an error indicates a missing record
func IsNotFoundError(err error) bool {
	return errors.Is(err, pgx.ErrNoRows) || errors.Is(err, ErrNotFound)
}

// NewError converts native pgx/postgres driver faults into domain apperr classifications.
func NewError(err error, scope Scope) error {
	if err == nil {
		return nil
	}

	// Don't intercept app errors
	var appErr *apperr.Error
	if errors.As(err, &appErr) {
		return err
	}

	// Intercept "Not Found" exceptions
	if IsNotFoundError(err) {
		return apperr.NewNotFound(err, fmt.Sprintf("%s could not be found.", scope))
	}

	// Inspect specific structural PostgreSQL constraints
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			return apperr.NewConflict(err, fmt.Sprintf("A conflict occurred. This %s already exists.", scope))
		case "23503": // foreign_key_violation
			return apperr.NewInvalidInput(err, fmt.Sprintf("A referenced %s record does not exist.", scope))
		case "23502": // not_null_violation
			return apperr.NewInvalidInput(err, "A required field is missing.")
		case "23514": // check_violation
			return apperr.NewInvalidInput(err, "The provided data failed validation rules.")
		case "22001": // string_data_right_truncation
			return apperr.NewInvalidInput(err, "A provided text field exceeds the maximum allowed length.")
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

	// Default fallback
	return apperr.NewInternal(err, "")
}
