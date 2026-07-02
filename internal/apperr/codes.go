package apperr

import (
	"net/http"
	"strings"
)

type Code string

const (
	CodeBadRequest           Code = "BAD_REQUEST"            // 400 Bad Request
	CodeInvalidInput         Code = "INVALID_INPUT"          // 400 Bad Request (Validation specific)
	CodeUnauthorized         Code = "UNAUTHORIZED"           // 401 Unauthorized (Fallback alignment)
	CodeForbidden            Code = "FORBIDDEN"              // 403 Forbidden (Authenticated, but lacked permissions)
	CodeNotFound             Code = "NOT_FOUND"              // 404 Not Found
	CodeMethodNotAllowed     Code = "METHOD_NOT_ALLOWED"     // 405 Method Not Allowed
	CodeConflict             Code = "CONFLICT"               // 409 Conflict (State or resource duplication)
	CodeGone                 Code = "GONE"                   // 410 Gone (Permanently deleted)
	CodePreconditionFailed   Code = "PRECONDITION_FAILED"    // 412 Precondition Failed (ETag/Optimistic concurrency match failure)
	CodePayloadTooLarge      Code = "PAYLOAD_TOO_LARGE"      // 413 Payload Too Large
	CodeUnsupportedMediaType Code = "UNSUPPORTED_MEDIA_TYPE" // 415 Unsupported Media Type
	CodeUnprocessableEntity  Code = "UNPROCESSABLE_ENTITY"   // 422 Unprocessable Entity (Semantic business logic rules)
	CodeTooManyRequests      Code = "TOO_MANY_REQUESTS"      // 429 Too Many Requests (Rate limiting triggered)
	CodeInternal             Code = "INTERNAL"               // 500 Internal Server Error
	CodeNotImplemented       Code = "NOT_IMPLEMENTED"        // 501 Not Implemented
	CodeBadGateway           Code = "BAD_GATEWAY"            // 502 Bad Gateway (Downstream third-party/microservice failure)
	CodeServiceUnavailable   Code = "SERVICE_UNAVAILABLE"    // 503 Service Unavailable
	CodeGatewayTimeout       Code = "GATEWAY_TIMEOUT"        // 504 Gateway Timeout (Downstream third-party/microservice timeout)
	CodeRequestTimeout       Code = "REQUEST_TIMEOUT"        // 408 Request Timeout (Your request/DB operation deadline expired)
	CodeClientClosedRequest  Code = "CLIENT_CLOSED_REQUEST"  // 499 Client Closed Request (Non-standard Nginx/Envoy disconnect)
)

type codeMetadata struct {
	status      int
	title       string
	description string
}

// codesRegistry organizes code values
var codesRegistry = map[Code]codeMetadata{
	CodeBadRequest: {
		status:      http.StatusBadRequest,
		title:       "Bad Request",
		description: "The request payload or syntax is malformed.",
	},
	CodeInvalidInput: {
		status:      http.StatusBadRequest,
		title:       "Invalid Input Data",
		description: "One or more fields failed validation rules.",
	},
	CodeUnauthorized: {
		status:      http.StatusUnauthorized,
		title:       "Unauthorized Access",
		description: "The provided credentials are invalid or expired.",
	},
	CodeForbidden: {
		status:      http.StatusForbidden,
		title:       "Permission Denied",
		description: "You lack the required permissions for this action.",
	},
	CodeNotFound: {
		status:      http.StatusNotFound,
		title:       "Resource Not Found",
		description: "The requested resource could not be found.",
	},
	CodeMethodNotAllowed: {
		status:      http.StatusMethodNotAllowed,
		title:       "Method Not Allowed",
		description: "The HTTP method is not supported for this path.",
	},
	CodeConflict: {
		status:      http.StatusConflict,
		title:       "Resource Conflict",
		description: "The operation conflicted with the current state of a resource.",
	},
	CodeGone: {
		status:      http.StatusGone,
		title:       "Resource No Longer Available",
		description: "The requested resource has been permanently deleted.",
	},
	CodePreconditionFailed: {
		status:      http.StatusPreconditionFailed,
		title:       "Precondition Failed",
		description: "Target resource state has changed. Please refresh and retry.",
	},
	CodePayloadTooLarge: {
		status:      http.StatusRequestEntityTooLarge,
		title:       "Payload Too Large",
		description: "The request body exceeds the maximum size limit.",
	},
	CodeUnsupportedMediaType: {
		status:      http.StatusUnsupportedMediaType,
		title:       "Unsupported Media Type",
		description: "Content-Type must be application/json.",
	},
	CodeUnprocessableEntity: {
		status:      http.StatusUnprocessableEntity,
		title:       "Unprocessable Entity",
		description: "The request is valid but breaks semantic business logic rules.",
	},
	CodeTooManyRequests: {
		status:      http.StatusTooManyRequests,
		title:       "Too Many Requests",
		description: "Rate limit exceeded. Please slow down.",
	},
	CodeInternal: {
		status:      http.StatusInternalServerError,
		title:       "Internal Server Error",
		description: "An unexpected condition occurred on our servers.",
	},
	CodeNotImplemented: {
		status:      http.StatusInternalServerError,
		title:       "Feature Not Implemented",
		description: "This server capability is not yet supported.",
	},
	CodeBadGateway: {
		status:      http.StatusBadGateway,
		title:       "Bad Gateway",
		description: "An upstream dependency returned an invalid response.",
	},
	CodeServiceUnavailable: {
		status:      http.StatusInternalServerError,
		title:       "Service Temporarily Unavailable",
		description: "The server is temporarily down for maintenance or overloaded.",
	},
	CodeGatewayTimeout: {
		status:      http.StatusGatewayTimeout,
		title:       "Gateway Timeout",
		description: "An upstream dependency failed to respond in time.",
	},
	CodeRequestTimeout: {
		status:      http.StatusRequestTimeout,
		title:       "Request Timeout",
		description: "The execution timeout deadline was exceeded.",
	},
	CodeClientClosedRequest: {
		status:      499,
		title:       "Client Closed Connection",
		description: "The client disconnected before processing completed.",
	},
}

// Status returns the error HTTP status code.
func (c Code) Status() int {
	if meta, ok := codesRegistry[c]; ok {
		return meta.status
	}
	return http.StatusInternalServerError
}

// Title returns the generic error title.
func (c Code) Title() string {
	if meta, ok := codesRegistry[c]; ok {
		return meta.title
	}
	return "An Unexpected Error Occurred"
}

// Description returns the generic error description.
func (c Code) Description() string {
	if meta, ok := codesRegistry[c]; ok {
		return meta.description
	}
	return "An internal server error occurred while processing your request."
}

// Slug transforms the code string into a lowercase URL.
func (c Code) Slug() string {
	return strings.ToLower(strings.ReplaceAll(string(c), "_", "-"))
}
