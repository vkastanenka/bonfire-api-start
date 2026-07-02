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
		return apperr.NewNotFound(err, fmt.Sprintf("%s could not be found.", name))
	}

	// Inspect specific structural PostgreSQL constraints
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			return apperr.NewConflict(err, fmt.Sprintf("A conflict occurred. This %s already exists.", name))
		case "23503": // foreign_key_violation
			return apperr.NewInvalidInput(err, fmt.Sprintf("A referenced %s record does not exist.", name))
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
