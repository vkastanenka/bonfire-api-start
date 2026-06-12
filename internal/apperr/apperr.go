package apperr

import (
	"fmt"
)

// Type defines the category of application error.
type Type string

const (
	TypeInternal         Type = "INTERNAL"
	TypeInvalidInput     Type = "INVALID_INPUT"
	TypePayloadTooLarge  Type = "PAYLOAD_TOO_LARGE"
	TypeNotFound         Type = "NOT_FOUND"
	TypeConflict         Type = "CONFLICT"
	TypeUnauthenticated  Type = "UNAUTHENTICATED"
	TypeMethodNotAllowed Type = "METHOD_NOT_ALLOWED"
	TypeTooManyRequests  Type = "TOO_MANY_REQUESTS"
	TypeBadRequest       Type = "BAD_REQUEST"
)

// Error represents a structured domain error.
type Error struct {
	Type    Type              `json:"-"`
	Message string            `json:"message"`
	Details map[string]string `json:"-"`
	Err     error             `json:"-"`
}

// Value receiver prevents typed nil pointer bugs
func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Value receiver allows standard errors.Unwrap to function seamlessly
func (e *Error) Unwrap() error {
	return e.Err
}

// Option defines a function signature for configuring an Error.
type Option func(*Error)

// WithErr wraps an underlying cause error.
func WithErr(err error) Option {
	return func(e *Error) {
		e.Err = err
	}
}

// WithDetails attaches key-value metadata to the error.
func WithDetails(details map[string]string) Option {
	return func(e *Error) {
		e.Details = details
	}
}

// Generic factory to minimize boilerplate
func newError(t Type, msg string, opts ...Option) *Error {
	err := &Error{
		Type:    t,
		Message: msg,
	}
	for _, opt := range opts {
		opt(err)
	}
	return err
}

// Public Factory Functions
func NewInvalidInput(msg string, opts ...Option) *Error {
	return newError(TypeInvalidInput, msg, opts...)
}

func NewInternal(msg string, opts ...Option) *Error {
	return newError(TypeInternal, msg, opts...)
}

func NewNotFound(msg string, opts ...Option) *Error {
	return newError(TypeNotFound, msg, opts...)
}

func NewPayloadTooLarge(msg string, opts ...Option) *Error {
	return newError(TypePayloadTooLarge, msg, opts...)
}

func NewConflict(msg string, opts ...Option) *Error {
	return newError(TypeConflict, msg, opts...)
}

// TODO: Rename to NewUnauthorized
func NewUnauthenticated(msg string, opts ...Option) *Error {
	return newError(TypeUnauthenticated, msg, opts...)
}

func NewMethodNotAllowed(msg string, opts ...Option) *Error {
	return newError(TypeMethodNotAllowed, msg, opts...)
}

func NewTooManyRequests(msg string, opts ...Option) *Error {
	return newError(TypeTooManyRequests, msg, opts...)
}

func NewBadRequest(msg string, opts ...Option) *Error {
	return newError(TypeBadRequest, msg, opts...)
}
