package apperr

import (
	"fmt"
)

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

// Constants
const (
	BaseDocURL = "https://api.bonfire.com/errors"
)

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

// New initializes a structured domain error with fallbacks and custom options.
func New(code Code, detail string, opts ...ErrorOption) error {
	if detail == "" {
		detail = code.Description()
	}

	e := &Error{
		Code:   code,
		Detail: detail,
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

// Param appends a single invalid field validation context.
func Param(name, reason string) ErrorOption {
	return func(e *Error) {
		e.InvalidParams = append(e.InvalidParams, InvalidParam{Name: name, Reason: reason})
	}
}

// Params appends a pre-collected slice of invalid fields.
func Params(params []InvalidParam) ErrorOption {
	return func(e *Error) {
		if len(params) > 0 {
			e.InvalidParams = append(e.InvalidParams, params...)
		}
	}
}
