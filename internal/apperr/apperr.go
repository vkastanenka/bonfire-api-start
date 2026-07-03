package apperr

import (
	"fmt"
)

// InvalidParam represents a single field validation failure.
type InvalidParam struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

// Error represents a structured domain-specific application failure.
type Error struct {
	Code          Code           `json:"code"`
	Detail        string         `json:"detail"`
	InvalidParams []InvalidParam `json:"invalid_params,omitempty"`
	Err           error          `json:"-"`
}

// ErrorOption defines a functional option configuration pattern.
type ErrorOption func(*Error)

// Assert compile-time correctness for the native error interface assignment.
var _ error = (*Error)(nil)

// Error converts the domain error into a descriptive string for logging engines.
func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Detail, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Detail)
}

// Unwrap handles nested native standard error unbundling chains
func (e *Error) Unwrap() error { return e.Err }

// Is implements Go standard errors.Is comparison
func (e *Error) Is(target error) bool {
	if target == nil {
		return false
	}

	if t, ok := target.(*Error); ok {
		return e.Code == t.Code
	}

	return false
}

// build initializes a structured domain error with fallbacks and custom options.
func build(code Code, err error, detail string, opts ...ErrorOption) error {
	if detail == "" {
		detail = code.Detail()
	}

	e := &Error{
		Code:   code,
		Detail: detail,
		Err:    err,
	}

	for _, opt := range opts {
		opt(e)
	}
	return e
}

// Err wraps a lower-level root cause error.
func Err(err error) ErrorOption {
	return func(e *Error) {
		if err != nil {
			e.Err = err
		}
	}
}

// Param constructs and appends a single invalid field validation context.
func Param(name, reason string) ErrorOption {
	return func(e *Error) {
		e.InvalidParams = append(e.InvalidParams, InvalidParam{
			Name:   name,
			Reason: reason,
		})
	}
}

// Params appends a pre-collected slice of invalid fields directly.
func Params(params []InvalidParam) ErrorOption {
	return func(e *Error) {
		if len(params) > 0 {
			e.InvalidParams = append(e.InvalidParams, params...)
		}
	}
}
