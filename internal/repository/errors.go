package repository

import (
	"errors"

	"github.com/jackc/pgx/v5"
)

// ErrNotFound is a driver-agnostic sentinel for missing rows
var ErrNotFound = errors.New("resource not found")

// IsNotFoundError checks if an error indicates a missing record
func IsNotFoundError(err error) bool {
	return errors.Is(err, pgx.ErrNoRows) || errors.Is(err, ErrNotFound)
}
