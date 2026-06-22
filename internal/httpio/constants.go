package httpio

const (
	// Header Errors
	UnsupportedMediaTypeMsg = "Missing or invalid Content-Type header; must be application/json."

	// Client/Stream Errors
	ClientClosedConnectionMsg = "Client closed connection mid-request."
	PayloadTooLargeMsg        = "Request body exceeds 1MB limit."
	EmptyBodyMsg              = "Request body cannot be empty."

	// JSON Structural Errors
	MalformedJSONMsg       = "Malformed request body JSON syntax."
	TruncatedJSONMsg       = "Truncated or malformed JSON structure received."
	InvalidPayloadMsg      = "Malformed or invalid request body JSON payload."
	SingleValueRequiredMsg = "Request body must contain only a single JSON value."

	// Field Validation & Typing Errors
	InvalidFieldTypeMsg     = "Invalid data type provided for request body field(s)."
	FieldTypeExpectationFmt = "Must be of type %s"
	UnknownFieldFmt         = "Unknown field '%s' present in request body."

	// Internal Errors
	DecodeErrMsg     = "An unexpected parsing error occurred."
	ReqTimeoutMsg    = "Request timed out."
	HTTPReqFailedMsg = "http request failed"
)

// SuccessResponse defines the standard envelope for all successful API responses.
type SuccessResponse[T any] struct {
	Message string `json:"message,omitempty"` // Optional human-readable message
	Data    T      `json:"data"`              // The actual payload
	Meta    any    `json:"meta,omitempty"`    // Pagination, cursors, or extra context
}

// OffsetPagination defines the meta payload for page-based lists.
type OffsetPagination struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	TotalItems int `json:"total_items"`
	TotalPages int `json:"total_pages"`
}

// CursorPagination defines the metadata returned for keyset-paginated lists.
type CursorPagination struct {
	NextCursor *string `json:"next_cursor,omitempty"` // Opaque string for the client
	PageSize   int32   `json:"page_size"`             // Count of items in this specific batch
}
