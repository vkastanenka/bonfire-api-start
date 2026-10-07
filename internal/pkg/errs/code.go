package errs

// Code represents standard gRPC-aligned error codes used to categorize errors
// across system boundaries (https://github.com/googleapis/googleapis/blob/master/google/rpc/code.proto).
type Code int

const (
	CodeOK Code = iota
	CodeCancelled
	CodeUnknown
	CodeInvalidArgument
	CodeDeadlineExceeded
	CodeNotFound
	CodeAlreadyExists
	CodePermissionDenied
	CodeResourceExhausted
	CodeFailedPrecondition
	CodeAborted
	CodeOutOfRange
	CodeUnimplemented
	CodeInternal
	CodeUnavailable
	CodeDataLoss
	CodeUnauthenticated
	codeMax
)

// codeNames maps Code enum values to string representations.
var codeNames = [...]string{
	CodeOK:                 "OK",
	CodeCancelled:          "CANCELLED",
	CodeUnknown:            "UNKNOWN",
	CodeInvalidArgument:    "INVALID_ARGUMENT",
	CodeDeadlineExceeded:   "DEADLINE_EXCEEDED",
	CodeNotFound:           "NOT_FOUND",
	CodeAlreadyExists:      "ALREADY_EXISTS",
	CodePermissionDenied:   "PERMISSION_DENIED",
	CodeResourceExhausted:  "RESOURCE_EXHAUSTED",
	CodeFailedPrecondition: "FAILED_PRECONDITION",
	CodeAborted:            "ABORTED",
	CodeOutOfRange:         "OUT_OF_RANGE",
	CodeUnimplemented:      "UNIMPLEMENTED",
	CodeInternal:           "INTERNAL",
	CodeUnavailable:        "UNAVAILABLE",
	CodeDataLoss:           "DATA_LOSS",
	CodeUnauthenticated:    "UNAUTHENTICATED",
}

// codeMessages maps Code enum values to default human-readable descriptions.
var codeMessages = [...]string{
	CodeOK:                 "The operation completed successfully.",
	CodeCancelled:          "The operation was cancelled.",
	CodeUnknown:            "An unknown error occurred.",
	CodeInvalidArgument:    "The client specified an invalid argument.",
	CodeDeadlineExceeded:   "The deadline expired before the operation could complete.",
	CodeNotFound:           "The requested resource was not found.",
	CodeAlreadyExists:      "The resource that the client attempted to create already exists.",
	CodePermissionDenied:   "The caller does not have permission to execute the specified operation.",
	CodeResourceExhausted:  "A resource limit or quota has been exceeded.",
	CodeFailedPrecondition: "The operation was rejected because the system is not in a state required for execution.",
	CodeAborted:            "The operation was aborted.",
	CodeOutOfRange:         "The operation was attempted past the valid range.",
	CodeUnimplemented:      "The operation is not implemented or supported in this service.",
	CodeInternal:           "An internal error occurred.",
	CodeUnavailable:        "The service is currently unavailable. Please try again later.",
	CodeDataLoss:           "Unrecoverable data loss or corruption occurred.",
	CodeUnauthenticated:    "The request does not have valid authentication credentials for the operation.",
}

// codeHTTPStatuses maps Code enum values to corresponding HTTP status codes.
var codeHTTPStatuses = [...]int{
	CodeOK:                 200,
	CodeCancelled:          499,
	CodeUnknown:            500,
	CodeInvalidArgument:    400,
	CodeDeadlineExceeded:   504,
	CodeNotFound:           404,
	CodeAlreadyExists:      409,
	CodePermissionDenied:   403,
	CodeResourceExhausted:  429,
	CodeFailedPrecondition: 400,
	CodeAborted:            409,
	CodeOutOfRange:         400,
	CodeUnimplemented:      501,
	CodeInternal:           500,
	CodeUnavailable:        503,
	CodeDataLoss:           500,
	CodeUnauthenticated:    401,
}

// IsValid returns true if code falls within valid enum range [CodeOK, codeMax).
func (c Code) IsValid() bool {
	return c >= CodeOK && c < codeMax
}

// Name returns string identifier for code (e.g., "INVALID_ARGUMENT").
// Returns "UNKNOWN" if invalid.
func (c Code) Name() string {
	if c.IsValid() {
		return codeNames[c]
	}
	return codeNames[CodeUnknown]
}

// Message returns default descriptive message for code.
// Returns default unknown message if invalid.
func (c Code) Message() string {
	if c.IsValid() {
		return codeMessages[c]
	}
	return codeMessages[CodeUnknown]
}

// HTTPStatus maps code to corresponding HTTP status code (e.g., 400 for CodeInvalidArgument).
// Returns 500 if invalid.
func (c Code) HTTPStatus() int {
	if c.IsValid() {
		return codeHTTPStatuses[c]
	}
	return codeHTTPStatuses[CodeUnknown]
}
