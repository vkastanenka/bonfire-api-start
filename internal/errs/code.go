package errs

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"unsafe"
)

// https://github.com/googleapis/googleapis/blob/master/google/rpc/code.proto

var ErrInvalidCode = errors.New("apperr: invalid error code")

type Code int32

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
	CodeOK:                 "Success.",
	CodeCancelled:          "Request cancelled.",
	CodeUnknown:            "An unexpected error occurred.",
	CodeInvalidArgument:    "Invalid input provided.",
	CodeDeadlineExceeded:   "Request timed out.",
	CodeNotFound:           "Resource not found.",
	CodeAlreadyExists:      "Resource already exists.",
	CodePermissionDenied:   "Permission denied.",
	CodeResourceExhausted:  "Rate limit or quota exceeded.",
	CodeFailedPrecondition: "System state prevents execution.",
	CodeAborted:            "Operation aborted.",
	CodeOutOfRange:         "Value out of valid range.",
	CodeUnimplemented:      "Feature not supported.",
	CodeInternal:           "Internal server error.",
	CodeUnavailable:        "Service temporarily unavailable.",
	CodeDataLoss:           "Data loss occurred.",
	CodeUnauthenticated:    "Authentication required.",
}

func Parse(raw string) (Code, error) {
	if raw == "" {
		return CodeUnknown, ErrInvalidCode
	}
	b := unsafe.Slice(unsafe.StringData(raw), len(raw))
	return ParseBytes(b)
}

func ParseBytes(b []byte) (Code, error) {
	b = bytes.TrimSpace(b)
	if len(b) == 0 {
		return CodeUnknown, ErrInvalidCode
	}

	for i := 0; i < int(codeMax); i++ {
		nameBytes := unsafe.Slice(unsafe.StringData(codeNames[i]), len(codeNames[i]))
		if bytes.EqualFold(nameBytes, b) {
			return Code(i), nil
		}
	}

	if n, err := strconv.ParseInt(unsafe.String(unsafe.SliceData(b), len(b)), 10, 32); err == nil {
		c := Code(n)
		if c.IsValid() {
			return c, nil
		}
	}

	return CodeUnknown, ErrInvalidCode
}

func (c Code) IsValid() bool {
	return c >= CodeOK && c < codeMax
}

func (c Code) String() string {
	if c.IsValid() {
		return codeNames[c]
	}
	return fmt.Sprintf("CODE_%d", c)
}

func (c Code) Message() string {
	if c.IsValid() {
		return codeMessages[c]
	}
	return "An unknown error occurred."
}

func (c Code) HTTPStatus() int {
	switch c {
	case CodeOK:
		return 200
	case CodeInvalidArgument, CodeOutOfRange:
		return 400
	case CodeUnauthenticated:
		return 401
	case CodePermissionDenied:
		return 403
	case CodeNotFound:
		return 404
	case CodeAlreadyExists, CodeAborted:
		return 409
	case CodeFailedPrecondition:
		return 412
	case CodeResourceExhausted:
		return 429
	case CodeCancelled:
		return 499
	case CodeUnimplemented:
		return 501
	case CodeUnavailable:
		return 503
	case CodeDeadlineExceeded:
		return 504
	default:
		return 500
	}
}

func (c Code) MarshalText() ([]byte, error) {
	return []byte(c.String()), nil
}

func (c *Code) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		*c = CodeOK
		return nil
	}

	parsed, err := ParseBytes(text)
	if err != nil {
		return err
	}
	*c = parsed
	return nil
}

func (c Code) MarshalJSON() ([]byte, error) {
	s := c.String()
	b := make([]byte, 0, len(s)+2)
	b = append(b, '"')
	b = append(b, s...)
	b = append(b, '"')
	return b, nil
}

func (c *Code) UnmarshalJSON(b []byte) error {
	if len(b) == 0 || bytes.Equal(b, []byte("null")) {
		*c = CodeOK
		return nil
	}

	if b[0] == '"' && b[len(b)-1] == '"' {
		return c.UnmarshalText(b[1 : len(b)-1])
	}

	if n, err := strconv.ParseInt(string(b), 10, 32); err == nil {
		code := Code(n)
		if code.IsValid() {
			*c = code
			return nil
		}
	}

	return ErrInvalidCode
}
