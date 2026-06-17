package apperr

import (
	"errors"
	"fmt"
)

// ValidationError represents specific field-level validation issues
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Error represents a structured domain error
type Error struct {
	Code             Code              `json:"code"`
	Message          string            `json:"message"`
	Details          map[string]any    `json:"details,omitempty"`
	ValidationErrors []ValidationError `json:"validation_errors,omitempty"`
	Timestamp        string            `json:"timestamp,omitempty"`
	RequestID        string            `json:"request_id,omitempty"`
	TraceID          string            `json:"trace_id,omitempty"`
	Err              error             `json:"-"` // Explicitly ignore the internal error during JSON serialization
}

// ErrorResponse is the source of truth for the JSON API contract
type ErrorResponse struct {
	Code             string            `json:"code"`
	Message          string            `json:"message,omitempty"`
	Details          map[string]any    `json:"details,omitempty"`
	ValidationErrors []ValidationError `json:"validation_errors,omitempty"`
	Timestamp        string            `json:"timestamp,omitempty"`
	RequestID        string            `json:"request_id,omitempty"`
	TraceID          string            `json:"trace_id,omitempty"`
}

// Error implements the standard error interface.
var _ error = (*Error)(nil)

// New creates a new domain error
func New(code Code, msg string, opts ...Option) error {
	err := &Error{
		Code:    code,
		Message: msg,
	}
	for _, opt := range opts {
		opt(err)
	}
	return err
}

// Error implements the error interface.
// Using a pointer receiver is correct here, but we ensure safe creation via the New() function.
func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap allows errors.Is and errors.As
func (e *Error) Unwrap() error {
	return e.Err
}

// Option defines a function signature for configuring an Error
type Option func(*Error)

// WithErr wraps an underlying cause error
func WithErr(err error) Option {
	return func(e *Error) {
		e.Err = err
	}
}

// WithDetails attaches key-value metadata to the error
func WithDetails(key string, value any) Option {
	return func(e *Error) {
		if e.Details == nil {
			e.Details = make(map[string]any)
		}
		e.Details[key] = value
	}
}

// WithValidationErr appends a single field validation error
func WithValidationErr(field, message string) Option {
	return func(e *Error) {
		e.ValidationErrors = append(e.ValidationErrors, ValidationError{
			Field:   field,
			Message: message,
		})
	}
}

// WithValidationErrors appends a slice of validation errors
func WithValidationErrors(errs []ValidationError) Option {
	return func(e *Error) {
		e.ValidationErrors = append(e.ValidationErrors, errs...)
	}
}

// ErrorCode safely extracts the application code from any error
func ErrorCode(err error) Code {
	if err == nil {
		return ""
	}

	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr.Code
	}

	// If it's a standard Go error not wrapped in apperr, default to Internal
	return CodeInternal
}

// IsCode allows for quick comparisons
func (e *Error) IsCode(c Code) bool {
	return e.Code == c
}

// ToResponse formats into response JSON
func (e *Error) ToResponse() ErrorResponse {
	return ErrorResponse{
		Code:             string(e.Code),
		Message:          e.Message,
		Details:          e.Details,
		ValidationErrors: e.ValidationErrors,
		Timestamp:        e.Timestamp,
		RequestID:        e.RequestID,
		TraceID:          e.TraceID,
	}
}
