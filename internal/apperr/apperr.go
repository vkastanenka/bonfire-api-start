package apperr

import (
	"errors"
	"fmt"
)

// Code defines the category of application error
type Code string

const (
	CodeInternal             Code = "INTERNAL"
	CodeInvalidInput         Code = "INVALID_INPUT"
	CodePayloadTooLarge      Code = "PAYLOAD_TOO_LARGE"
	CodeNotFound             Code = "NOT_FOUND"
	CodeConflict             Code = "CONFLICT"
	CodeUnauthenticated      Code = "UNAUTHENTICATED"
	CodeMethodNotAllowed     Code = "METHOD_NOT_ALLOWED"
	CodeTooManyRequests      Code = "TOO_MANY_REQUESTS"
	CodeBadRequest           Code = "BAD_REQUEST"
	CodeUnsupportedMediaType Code = "UNSUPPORTED_MEDIA_TYPE"
)

// Error represents a structured domain error
type Error struct {
	Code    Code
	Message string
	Details map[string]any
	Err     error
}

// Error implements the standard error interface.
var _ error = (*Error)(nil)

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
