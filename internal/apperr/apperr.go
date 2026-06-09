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
	Type    Type   `json:"-"`
	Message string `json:"message"`
	Err     error  `json:"-"` // Underlying root cause error
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

// Helper factory functions to clean up creation
func NewInvalidInput(msg string, err error) *AppError {
	return &AppError{Type: TypeInvalidInput, Message: msg, Err: err}
}

func NewInternal(msg string, err error) *AppError {
	return &AppError{Type: TypeInternal, Message: msg, Err: err}
}

func NewNotFound(msg string, err error) *AppError {
	return &AppError{Type: TypeNotFound, Message: msg, Err: err}
}

func NewPayloadTooLarge(msg string, err error) *AppError {
	return &AppError{Type: TypePayloadTooLarge, Message: msg, Err: err}
}
