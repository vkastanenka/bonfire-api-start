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
	DecodeErrMsg = "An unexpected parsing error occurred"
)
