package apperr

import "net/http"

// Code defines the category of application error
type Code string

const (
	// 400s: Client-side issues
	CodeBadRequest           Code = "BAD_REQUEST"
	CodeInvalidInput         Code = "INVALID_INPUT"
	CodePayloadTooLarge      Code = "PAYLOAD_TOO_LARGE"
	CodeUnsupportedMediaType Code = "UNSUPPORTED_MEDIA_TYPE"

	// 401/403: Authentication & Authorization
	CodeUnauthenticated Code = "UNAUTHENTICATED"
	CodeUnauthorized    Code = "UNAUTHORIZED" // Distinction: Not logged in vs no permission

	// 405: Method not allowed
	CodeMethodNotAllowed Code = "METHOD_NOT_ALLOWED"

	// 404/409: Resource State
	CodeNotFound Code = "NOT_FOUND"
	CodeConflict Code = "CONFLICT"
	CodeGone     Code = "GONE" // Useful for deleted resources

	// 429: Rate Limiting
	CodeTooManyRequests Code = "TOO_MANY_REQUESTS"

	// 500s: Server-side issues
	CodeInternal           Code = "INTERNAL"
	CodeNotImplemented     Code = "NOT_IMPLEMENTED"     // For unfinished endpoints
	CodeServiceUnavailable Code = "SERVICE_UNAVAILABLE" // For dependency failures

	// Connectivity
	CodeRequestTimeout      Code = "REQUEST_TIMEOUT"
	CodeClientClosedRequest Code = "CLIENT_CLOSED_REQUEST"
)

func (c Code) HTTPStatus() int {
	switch c {
	// 400s: Client-side issues
	case CodeBadRequest:
		return http.StatusBadRequest
	case CodeInvalidInput:
		return http.StatusBadRequest
	case CodePayloadTooLarge:
		return http.StatusRequestEntityTooLarge
	case CodeUnsupportedMediaType:
		return http.StatusUnsupportedMediaType

	// 401/403: Authentication & Authorization
	case CodeUnauthenticated:
		return http.StatusUnauthorized
	case CodeUnauthorized:
		return http.StatusForbidden

	// 405
	case CodeMethodNotAllowed:
		return http.StatusMethodNotAllowed

	// 404/409: Resource State
	case CodeNotFound:
		return http.StatusNotFound
	case CodeConflict:
		return http.StatusConflict
	case CodeGone:
		return http.StatusGone

	// 429
	case CodeTooManyRequests:
		return http.StatusTooManyRequests

	// 500s: Server-side issues
	case CodeInternal:
		return http.StatusInternalServerError
	case CodeNotImplemented:
		return http.StatusNotImplemented
	case CodeServiceUnavailable:
		return http.StatusServiceUnavailable

	// Connectivity
	case CodeRequestTimeout:
		return http.StatusRequestTimeout
	case CodeClientClosedRequest:
		// 499 is a non-standard but common Nginx status code for
		// "Client Closed Request"
		return 499

	// Default fallback
	default:
		return http.StatusInternalServerError
	}
}
