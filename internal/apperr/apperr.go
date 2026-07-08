package apperr

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

type Code string

type InvalidParam struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

type Error struct {
	Code          Code           `json:"code"`
	Detail        string         `json:"detail"`
	InvalidParams []InvalidParam `json:"invalid_params,omitempty"`
	Err           error          `json:"-"`
}

type ErrorOption func(*Error)

var _ error = (*Error)(nil)

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Detail, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Detail)
}

func (e *Error) Unwrap() error { return e.Err }

func (e *Error) Is(target error) bool {
	if target == nil {
		return false
	}
	if t, ok := target.(*Error); ok {
		return e.Code == t.Code
	}
	return false
}

const (
	CodeBadRequest           Code = "BAD_REQUEST"            // 400 Bad Request
	CodeInvalidInput         Code = "INVALID_INPUT"          // 400 Bad Request (Validation specific)
	CodeUnauthorized         Code = "UNAUTHORIZED"           // 401 Unauthorized
	CodeTokenExpired         Code = "TOKEN_EXPIRED"          // 401 Unauthorized
	CodeForbidden            Code = "FORBIDDEN"              // 403 Forbidden
	CodeNotFound             Code = "NOT_FOUND"              // 404 Not Found
	CodeMethodNotAllowed     Code = "METHOD_NOT_ALLOWED"     // 405 Method Not Allowed
	CodeConflict             Code = "CONFLICT"               // 409 Conflict
	CodeGone                 Code = "GONE"                   // 410 Gone
	CodePreconditionFailed   Code = "PRECONDITION_FAILED"    // 412 Precondition Failed
	CodePayloadTooLarge      Code = "PAYLOAD_TOO_LARGE"      // 413 Payload Too Large
	CodeUnsupportedMediaType Code = "UNSUPPORTED_MEDIA_TYPE" // 415 Unsupported Media Type
	CodeUnprocessableEntity  Code = "UNPROCESSABLE_ENTITY"   // 422 Unprocessable Entity
	CodeTooManyRequests      Code = "TOO_MANY_REQUESTS"      // 429 Too Many Requests
	CodeInternal             Code = "INTERNAL"               // 500 Internal Server Error
	CodeNotImplemented       Code = "NOT_IMPLEMENTED"        // 501 Not Implemented
	CodeBadGateway           Code = "BAD_GATEWAY"            // 502 Bad Gateway
	CodeServiceUnavailable   Code = "SERVICE_UNAVAILABLE"    // 503 Service Unavailable
	CodeGatewayTimeout       Code = "GATEWAY_TIMEOUT"        // 504 Gateway Timeout
	CodeRequestTimeout       Code = "REQUEST_TIMEOUT"        // 408 Request Timeout
	CodeClientClosedRequest  Code = "CLIENT_CLOSED_REQUEST"  // 499 Client Closed Request
)

type codeMetadata struct {
	status int
	title  string
	detail string
}

var codesRegistry = map[Code]codeMetadata{
	CodeBadRequest: {
		status: http.StatusBadRequest,
		title:  "Bad Request",
		detail: "The request payload or syntax is malformed.",
	},
	CodeInvalidInput: {
		status: http.StatusBadRequest,
		title:  "Invalid Input Data",
		detail: "One or more fields failed validation rules.",
	},
	CodeUnauthorized: {
		status: http.StatusUnauthorized,
		title:  "Unauthorized Access",
		detail: "The provided credentials are invalid or expired.",
	},
	CodeTokenExpired: {
		status: http.StatusUnauthorized,
		title:  "Token Expired",
		detail: "The token has expired and requires a refresh.",
	},
	CodeForbidden: {
		status: http.StatusForbidden,
		title:  "Permission Denied",
		detail: "You lack the required permissions for this action.",
	},
	CodeNotFound: {
		status: http.StatusNotFound,
		title:  "Resource Not Found",
		detail: "The requested resource could not be found.",
	},
	CodeMethodNotAllowed: {
		status: http.StatusMethodNotAllowed,
		title:  "Method Not Allowed",
		detail: "The HTTP method is not supported for this path.",
	},
	CodeConflict: {
		status: http.StatusConflict,
		title:  "Resource Conflict",
		detail: "The operation conflicted with the current state of a resource.",
	},
	CodeGone: {
		status: http.StatusGone,
		title:  "Resource No Longer Available",
		detail: "The requested resource has been permanently deleted.",
	},
	CodePreconditionFailed: {
		status: http.StatusPreconditionFailed,
		title:  "Precondition Failed",
		detail: "Target resource state has changed. Please refresh and retry.",
	},
	CodePayloadTooLarge: {
		status: http.StatusRequestEntityTooLarge,
		title:  "Payload Too Large",
		detail: "The request body exceeds the maximum size limit.",
	},
	CodeUnsupportedMediaType: {
		status: http.StatusUnsupportedMediaType,
		title:  "Unsupported Media Type",
		detail: "Content-Type must be application/json.",
	},
	CodeUnprocessableEntity: {
		status: http.StatusUnprocessableEntity,
		title:  "Unprocessable Entity",
		detail: "The request is valid but breaks semantic business logic rules.",
	},
	CodeTooManyRequests: {
		status: http.StatusTooManyRequests,
		title:  "Too Many Requests",
		detail: "Rate limit exceeded. Please slow down and retry shortly.",
	},
	CodeInternal: {
		status: http.StatusInternalServerError,
		title:  "Internal Server Error",
		detail: "An unexpected condition occurred on our servers.",
	},
	CodeNotImplemented: {
		status: http.StatusNotImplemented,
		title:  "Feature Not Implemented",
		detail: "This server capability is not yet supported.",
	},
	CodeBadGateway: {
		status: http.StatusBadGateway,
		title:  "Bad Gateway",
		detail: "An upstream dependency returned an invalid response.",
	},
	CodeServiceUnavailable: {
		status: http.StatusServiceUnavailable,
		title:  "Service Temporarily Unavailable",
		detail: "The server is temporarily down for maintenance or overloaded.",
	},
	CodeGatewayTimeout: {
		status: http.StatusGatewayTimeout,
		title:  "Gateway Timeout",
		detail: "An upstream dependency failed to respond in time.",
	},
	CodeRequestTimeout: {
		status: http.StatusRequestTimeout,
		title:  "Request Timeout",
		detail: "The execution timeout deadline was exceeded.",
	},
	CodeClientClosedRequest: {
		status: 499,
		title:  "Client Closed Connection",
		detail: "The client disconnected before processing completed.",
	},
}

func (c Code) Status() int {
	if m, ok := codesRegistry[c]; ok {
		return m.status
	}
	return http.StatusInternalServerError
}

func (c Code) Title() string {
	if m, ok := codesRegistry[c]; ok {
		return m.title
	}
	return "An Unexpected Error Occurred"
}

func (c Code) Detail() string {
	if m, ok := codesRegistry[c]; ok {
		return m.detail
	}
	return "An internal server error occurred."
}

func (c Code) Slug() string {
	return strings.ToLower(strings.ReplaceAll(string(c), "_", "-"))
}

func build(code Code, err error, detail string, opts ...ErrorOption) error {
	if detail == "" {
		detail = code.Detail()
	}
	e := &Error{Code: code, Detail: detail, Err: err}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

func Err(err error) ErrorOption {
	return func(e *Error) {
		if err != nil {
			e.Err = err
		}
	}
}

func Param(name, reason string) ErrorOption {
	return func(e *Error) { e.InvalidParams = append(e.InvalidParams, InvalidParam{Name: name, Reason: reason}) }
}

func Params(params []InvalidParam) ErrorOption {
	return func(e *Error) {
		if len(params) > 0 {
			e.InvalidParams = append(e.InvalidParams, params...)
		}
	}
}

func NewBadRequest(err error, detail string, opts ...ErrorOption) error {
	return build(CodeBadRequest, err, detail, opts...)
}
func NewInvalidInput(err error, detail string, opts ...ErrorOption) error {
	return build(CodeInvalidInput, err, detail, opts...)
}
func NewUnauthorized(err error, detail string, opts ...ErrorOption) error {
	return build(CodeUnauthorized, err, detail, opts...)
}
func NewTokenExpired(err error, detail string, opts ...ErrorOption) error {
	return build(CodeTokenExpired, err, detail, opts...)
}
func NewForbidden(err error, detail string, opts ...ErrorOption) error {
	return build(CodeForbidden, err, detail, opts...)
}
func NewNotFound(err error, detail string, opts ...ErrorOption) error {
	return build(CodeNotFound, err, detail, opts...)
}
func NewMethodNotAllowed(err error, detail string, opts ...ErrorOption) error {
	return build(CodeMethodNotAllowed, err, detail, opts...)
}
func NewConflict(err error, detail string, opts ...ErrorOption) error {
	return build(CodeConflict, err, detail, opts...)
}
func NewGone(err error, detail string, opts ...ErrorOption) error {
	return build(CodeGone, err, detail, opts...)
}
func NewPreconditionFailed(err error, detail string, opts ...ErrorOption) error {
	return build(CodePreconditionFailed, err, detail, opts...)
}
func NewPayloadTooLarge(err error, detail string, opts ...ErrorOption) error {
	return build(CodePayloadTooLarge, err, detail, opts...)
}
func NewUnsupportedMediaType(err error, detail string, opts ...ErrorOption) error {
	return build(CodeUnsupportedMediaType, err, detail, opts...)
}
func NewUnprocessableEntity(err error, detail string, opts ...ErrorOption) error {
	return build(CodeUnprocessableEntity, err, detail, opts...)
}
func NewTooManyRequests(err error, detail string, opts ...ErrorOption) error {
	return build(CodeTooManyRequests, err, detail, opts...)
}
func NewInternal(err error, detail string, opts ...ErrorOption) error {
	return build(CodeInternal, err, detail, opts...)
}
func NewNotImplemented(err error, detail string, opts ...ErrorOption) error {
	return build(CodeNotImplemented, err, detail, opts...)
}
func NewBadGateway(err error, detail string, opts ...ErrorOption) error {
	return build(CodeBadGateway, err, detail, opts...)
}
func NewServiceUnavailable(err error, detail string, opts ...ErrorOption) error {
	return build(CodeServiceUnavailable, err, detail, opts...)
}
func NewGatewayTimeout(err error, detail string, opts ...ErrorOption) error {
	return build(CodeGatewayTimeout, err, detail, opts...)
}
func NewRequestTimeout(err error, detail string, opts ...ErrorOption) error {
	return build(CodeRequestTimeout, err, detail, opts...)
}
func NewClientClosedRequest(err error, detail string, opts ...ErrorOption) error {
	return build(CodeClientClosedRequest, err, detail, opts...)
}

func IsCode(err error, code Code) bool {
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr.Code == code
	}
	return false
}

func IsBadRequest(err error) bool           { return IsCode(err, CodeBadRequest) }
func IsInvalidInput(err error) bool         { return IsCode(err, CodeInvalidInput) }
func IsUnauthorized(err error) bool         { return IsCode(err, CodeUnauthorized) }
func IsTokenExpired(err error) bool         { return IsCode(err, CodeTokenExpired) }
func IsForbidden(err error) bool            { return IsCode(err, CodeForbidden) }
func IsNotFound(err error) bool             { return IsCode(err, CodeNotFound) }
func IsMethodNotAllowed(err error) bool     { return IsCode(err, CodeMethodNotAllowed) }
func IsConflict(err error) bool             { return IsCode(err, CodeConflict) }
func IsGone(err error) bool                 { return IsCode(err, CodeGone) }
func IsPreconditionFailed(err error) bool   { return IsCode(err, CodePreconditionFailed) }
func IsPayloadTooLarge(err error) bool      { return IsCode(err, CodePayloadTooLarge) }
func IsUnsupportedMediaType(err error) bool { return IsCode(err, CodeUnsupportedMediaType) }
func IsUnprocessableEntity(err error) bool  { return IsCode(err, CodeUnprocessableEntity) }
func IsTooManyRequests(err error) bool      { return IsCode(err, CodeTooManyRequests) }
func IsInternal(err error) bool             { return IsCode(err, CodeInternal) }
func IsNotImplemented(err error) bool       { return IsCode(err, CodeNotImplemented) }
func IsBadGateway(err error) bool           { return IsCode(err, CodeBadGateway) }
func IsServiceUnavailable(err error) bool   { return IsCode(err, CodeServiceUnavailable) }
func IsGatewayTimeout(err error) bool       { return IsCode(err, CodeGatewayTimeout) }
func IsRequestTimeout(err error) bool       { return IsCode(err, CodeRequestTimeout) }
func IsClientClosedRequest(err error) bool  { return IsCode(err, CodeClientClosedRequest) }
