package apperr

import (
	"bonfire-api/internal/repository"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
)

// --- APPERR CONSTANTS ---

const (
	BaseDocURL = "https://api.bonfire.com/errors"
)

// --- APPERR TYPES ---

// InvalidParam
type InvalidParam struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

// Error
type Error struct {
	Code          Code
	Detail        string
	InvalidParams []InvalidParam
	Err           error
}

// ErrorOption
type ErrorOption func(*Error)

// --- APPERR COMPILE TIME ASSERTION ---

// Assert compile-time correctness for the error interface assignment
var _ error = (*Error)(nil)

// Error converts the internal model values to an explicit debugging line string for console logs
func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Detail, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Detail)
}

// Unwrap handles nested native standard error unbundling chains
func (e *Error) Unwrap() error { return e.Err }

// --- APPERR INITIALIZATION ---

// New
func New(code Code, detail string, opts ...ErrorOption) error {
	e := &Error{Code: code, Detail: detail}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// NewDBError
func NewDBError(err error, entityName ...string) error {
	if err == nil {
		return nil
	}

	entity := "Resource"
	if len(entityName) > 0 && entityName[0] != "" {
		entity = entityName[0]
	}

	// Intercept "Not Found" exceptions natively
	if repository.IsNotFoundError(err) {
		return New(CodeNotFound, fmt.Sprintf("%s could not be found.", entity), WithErr(err))
	}

	// Inspect specific structural PostgreSQL constraints
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			return New(CodeConflict, fmt.Sprintf("A conflict occurred. This %s already exists.", entity), WithErr(err))
		case "23503": // foreign_key_violation
			return New(CodeInvalidInput, "A referenced record does not exist.", WithErr(err))
		case "23502": // not_null_violation
			return New(CodeInvalidInput, "A required field is missing.", WithErr(err))
		case "23514": // check_violation
			return New(CodeInvalidInput, "The provided data failed validation rules.", WithErr(err))
		case "22001": // string_data_right_truncation
			return New(CodeInvalidInput, "A provided text field exceeds the maximum allowed length.", WithErr(err))
		case "22003": // numeric_value_out_of_range
			return New(CodeInvalidInput, "A provided number is out of the acceptable range.", WithErr(err))
		case "22P02": // invalid_text_representation (e.g., bad UUIDs)
			return New(CodeInvalidInput, "The data format is invalid or malformed.", WithErr(err))
		case "40001", "40P01": // serialization_failure & deadlock_detected
			return New(CodeConflict, "A resource conflict occurred. Please retry your request.", WithErr(err))
		case "57014": // query_canceled
			return New(CodeRequestTimeout, "The database operation timed out or was canceled.", WithErr(err))
		}
	}

	// Default fallback for unmapped infrastructure faults
	return New(CodeInternal, CodeInternal.Title(), WithErr(err))
}

// NewInternal
func NewInternal(err ...error) error {
	if len(err) > 0 && err[0] != nil {
		return New(CodeInternal, CodeInternal.Title(), WithErr(err[0]))
	}
	return New(CodeInternal, CodeInternal.Title())
}

// --- APPERR FUNCTIONS ---

// Is
func Is(err error, code Code) bool {
	if err == nil {
		return false
	}

	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr.Code == code
	}

	return false
}

// IsNotFound
func IsNotFound(err error) bool {
	return Is(err, CodeNotFound)
}

// WithErr couples lower-level execution context or database failures cleanly
func WithErr(err error) ErrorOption {
	return func(e *Error) {
		if err == nil {
			return
		}
		e.Err = err
	}
}

// WithInvalidParam appends a single distinct input parameters error to the parameter slice tracking validation bugs
func WithInvalidParam(name, reason string) ErrorOption {
	return func(e *Error) {
		e.InvalidParams = append(e.InvalidParams, InvalidParam{
			Name:   name,
			Reason: reason,
		})
	}
}

// WithInvalidParams chains whole structural groups of batch evaluation results directly
func WithInvalidParams(params []InvalidParam) ErrorOption {
	return func(e *Error) {
		e.InvalidParams = append(e.InvalidParams, params...)
	}
}
