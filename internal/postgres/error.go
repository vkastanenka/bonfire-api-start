package postgres

import (
	"errors"
	"fmt"

	"bonfire-api/internal/apperr"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// NewError converts native pgx/postgres driver faults into domain apperr classifications.
func NewError(entity Entity, err error) error {
	if err == nil {
		return nil
	}

	name := entity.String()

	// Intercept "Not Found" exceptions
	if errors.Is(err, pgx.ErrNoRows) {
		return apperr.NewNotFound(fmt.Sprintf("%s could not be found.", name), err)
	}

	// Inspect specific structural PostgreSQL constraints
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			return apperr.NewConflict(fmt.Sprintf("A conflict occurred. This %s already exists.", name), err)
		case "23503": // foreign_key_violation
			return apperr.NewInvalidInput(fmt.Sprintf("A referenced %s record does not exist.", name), err)
		case "23502": // not_null_violation
			return apperr.NewInvalidInput("A required field is missing.", err)
		case "23514": // check_violation
			return apperr.NewInvalidInput("The provided data failed validation rules.", err)
		case "22001": // string_data_right_truncation
			return apperr.NewInvalidInput("A provided text field exceeds the maximum allowed length.", err)
		case "22003": // numeric_value_out_of_range
			return apperr.NewInvalidInput("A provided number is out of the acceptable range.", err)
		case "22P02": // invalid_text_representation (e.g., bad UUIDs)
			return apperr.NewInvalidInput("The data format is invalid or malformed.", err)
		case "40001", "40P01": // serialization_failure & deadlock_detected
			return apperr.NewConflict("A resource conflict occurred. Please retry your request.", err)
		case "57014": // query_canceled
			return apperr.NewRequestTimeout("The database operation timed out or was canceled.", err)
		}
	}

	// Default fallback
	return apperr.NewInternal(err)
}
