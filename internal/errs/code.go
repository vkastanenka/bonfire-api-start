package errs

import (
	"bytes"
	"fmt"
	"strconv"
)

// https://github.com/googleapis/googleapis/blob/master/google/rpc/code.proto

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

var codeHTTPStatuses = [...]int{
	CodeOK:                 200,
	CodeInvalidArgument:    400,
	CodeOutOfRange:         400,
	CodeUnauthenticated:    401,
	CodePermissionDenied:   403,
	CodeNotFound:           404,
	CodeAlreadyExists:      409,
	CodeAborted:            409,
	CodeFailedPrecondition: 412,
	CodeResourceExhausted:  429,
	CodeCancelled:          499,
	CodeInternal:           500,
	CodeUnknown:            500,
	CodeDataLoss:           500,
	CodeUnimplemented:      501,
	CodeUnavailable:        503,
	CodeDeadlineExceeded:   504,
}

func Parse(raw int) (Code, error) {
	c := Code(raw)
	if !c.IsValid() {
		return CodeUnknown, fmt.Errorf("invalid error code: %d", raw)
	}
	return c, nil
}

func ParseString(s string) (Code, error) {
	for i, name := range codeNames {
		if name == s {
			return Code(i), nil
		}
	}

	if n, err := strconv.ParseInt(s, 10, 32); err == nil {
		c := Code(n)
		if c.IsValid() {
			return c, nil
		}
	}

	return CodeUnknown, fmt.Errorf("invalid error code string: %q", s)
}

func ParseBytes(b []byte) (Code, error) {
	if len(b) == 0 {
		return CodeOK, nil
	}
	return ParseString(string(b))
}

func (c Code) IsValid() bool {
	return c >= CodeOK && c < codeMax
}

func (c Code) Name() string {
	if int(c) < len(codeNames) {
		return codeNames[c]
	}
	return fmt.Sprintf("CODE_%d", c)
}

func (c Code) Message() string {
	if int(c) < len(codeMessages) {
		return codeMessages[c]
	}
	return "An internal error occurred."
}

func (c Code) HTTPStatus() int {
	if int(c) < len(codeHTTPStatuses) && codeHTTPStatuses[c] != 0 {
		return codeHTTPStatuses[c]
	}
	return 500
}

func (c Code) MarshalText() ([]byte, error) {
	return []byte(c.Name()), nil
}

func (c *Code) UnmarshalText(text []byte) error {
	parsed, err := ParseString(string(text))
	if err != nil {
		return err
	}
	*c = parsed
	return nil
}

func (c Code) MarshalJSON() ([]byte, error) {
	return strconv.AppendQuote(nil, c.Name()), nil
}

func (c *Code) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		*c = CodeOK
		return nil
	}

	parsed, err := ParseBytes(data)
	if err != nil {
		return err
	}

	*c = parsed
	return nil
}
