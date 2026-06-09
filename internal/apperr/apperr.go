package apperr

import (
	"fmt"
)

// Type defines the category of application error.
type Type string

const (
	TypeInternal        Type = "INTERNAL"
	TypeInvalidInput    Type = "INVALID_INPUT"
	TypePayloadTooLarge Type = "PAYLOAD_TOO_LARGE"
	TypeNotFound        Type = "NOT_FOUND"
	TypeConflict        Type = "CONFLICT"
	TypeUnauthenticated Type = "UNAUTHENTICATED"
)

// AppError represents a structured domain error.
type AppError struct {
	Type    Type              `json:"-"`
	Message string            `json:"message"`
	Details map[string]string `json:"-"`
	Err     error             `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// Option defines a function signature for configuring an AppError.
type Option func(*AppError)

// WithErr wraps an underlying cause error.
func WithErr(err error) Option {
	return func(e *AppError) {
		e.Err = err
	}
}

// WithDetails attaches key-value metadata to the error.
func WithDetails(details map[string]string) Option {
	return func(e *AppError) {
		e.Details = details
	}
}

// Generic factory to minimize boilerplate
func newAppError(t Type, msg string, opts ...Option) *AppError {
	err := &AppError{
		Type:    t,
		Message: msg,
	}
	for _, opt := range opts {
		opt(err)
	}
	return err
}

// Public Factory Functions
func NewInvalidInput(msg string, opts ...Option) *AppError {
	return newAppError(TypeInvalidInput, msg, opts...)
}
func NewInternal(msg string, opts ...Option) *AppError {
	return newAppError(TypeInternal, msg, opts...)
}
func NewNotFound(msg string, opts ...Option) *AppError {
	return newAppError(TypeNotFound, msg, opts...)
}
func NewPayloadTooLarge(msg string, opts ...Option) *AppError {
	return newAppError(TypePayloadTooLarge, msg, opts...)
}
func NewConflict(msg string, opts ...Option) *AppError {
	return newAppError(TypeConflict, msg, opts...)
}
func NewUnauthenticated(msg string, opts ...Option) *AppError {
	return newAppError(TypeUnauthenticated, msg, opts...)
}
