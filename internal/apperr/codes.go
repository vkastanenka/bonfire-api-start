package apperr

import (
	"net/http"
	"strings"
)

// --- CODE CONSTANTS ---

const (
	// 400s: Client-side issues
	CodeBadRequest           Code = "BAD_REQUEST"
	CodeInvalidInput         Code = "INVALID_INPUT"
	CodePayloadTooLarge      Code = "PAYLOAD_TOO_LARGE"
	CodeUnsupportedMediaType Code = "UNSUPPORTED_MEDIA_TYPE"

	// 401/403: Authentication & Authorization
	CodeUnauthenticated Code = "UNAUTHENTICATED"
	CodeUnauthorized    Code = "UNAUTHORIZED"

	// 405: Method not allowed
	CodeMethodNotAllowed Code = "METHOD_NOT_ALLOWED"

	// 404/409: Resource State
	CodeNotFound Code = "NOT_FOUND"
	CodeConflict Code = "CONFLICT"
	CodeGone     Code = "GONE"

	// 422: Unprocessable entity
	CodeUnprocessableEntity Code = "UNPROCESSABLE_ENTITY"

	// 429: Rate Limiting
	CodeTooManyRequests Code = "TOO_MANY_REQUESTS"

	// 500s: Server-side issues
	CodeInternal           Code = "INTERNAL"
	CodeNotImplemented     Code = "NOT_IMPLEMENTED"
	CodeServiceUnavailable Code = "SERVICE_UNAVAILABLE"

	// Connectivity
	CodeRequestTimeout      Code = "REQUEST_TIMEOUT"
	CodeClientClosedRequest Code = "CLIENT_CLOSED_REQUEST"
)

// --- CODE TYPES ---

// Code
type Code string

// --- CODE METHODS ---

// HTTPStatus returns the corresponding standard HTTP status code
func (c Code) HTTPStatus() int {
	switch c {
	case CodeBadRequest, CodeInvalidInput:
		return http.StatusBadRequest
	case CodePayloadTooLarge:
		return http.StatusRequestEntityTooLarge
	case CodeUnsupportedMediaType:
		return http.StatusUnsupportedMediaType
	case CodeUnauthenticated:
		return http.StatusUnauthorized
	case CodeUnauthorized:
		return http.StatusForbidden
	case CodeMethodNotAllowed:
		return http.StatusMethodNotAllowed
	case CodeNotFound:
		return http.StatusNotFound
	case CodeConflict:
		return http.StatusConflict
	case CodeGone:
		return http.StatusGone
	case CodeUnprocessableEntity:
		return http.StatusUnprocessableEntity
	case CodeTooManyRequests:
		return http.StatusTooManyRequests
	case CodeInternal, CodeNotImplemented, CodeServiceUnavailable:
		return http.StatusInternalServerError
	case CodeRequestTimeout:
		return http.StatusRequestTimeout
	case CodeClientClosedRequest:
		return 499 // Non-standard but common Nginx status for client disconnects
	default:
		return http.StatusInternalServerError
	}
}

// Title provides the generic, static description for the error classification
func (c Code) Title() string {
	switch c {
	case CodeBadRequest:
		return "Bad Request"
	case CodeInvalidInput:
		return "Invalid Input Data"
	case CodePayloadTooLarge:
		return "Payload Too Large"
	case CodeUnsupportedMediaType:
		return "Unsupported Media Type"
	case CodeUnauthenticated:
		return "Authentication Required"
	case CodeUnauthorized:
		return "Permission Denied"
	case CodeMethodNotAllowed:
		return "Method Not Allowed"
	case CodeNotFound:
		return "Resource Not Found"
	case CodeConflict:
		return "Resource Conflict"
	case CodeGone:
		return "Resource No Longer Available"
	case CodeUnprocessableEntity:
		return "Unprocessable Entity"
	case CodeTooManyRequests:
		return "Too Many Requests"
	case CodeInternal:
		return "Internal Server Error"
	case CodeNotImplemented:
		return "Feature Not Implemented"
	case CodeServiceUnavailable:
		return "Service Temporarily Unavailable"
	case CodeRequestTimeout:
		return "Request Timeout"
	case CodeClientClosedRequest:
		return "Client Closed Connection"
	default:
		return "An Unexpected Error Occurred"
	}
}

// Slug transforms the code string into a lowercase URL segment for docs linking
func (c Code) Slug() string {
	return strings.ToLower(strings.ReplaceAll(string(c), "_", "-"))
}
