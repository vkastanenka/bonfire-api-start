package errs

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
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

var codeBytes = [...][]byte{
	CodeOK:                 []byte("OK"),
	CodeCancelled:          []byte("CANCELLED"),
	CodeUnknown:            []byte("UNKNOWN"),
	CodeInvalidArgument:    []byte("INVALID_ARGUMENT"),
	CodeDeadlineExceeded:   []byte("DEADLINE_EXCEEDED"),
	CodeNotFound:           []byte("NOT_FOUND"),
	CodeAlreadyExists:      []byte("ALREADY_EXISTS"),
	CodePermissionDenied:   []byte("PERMISSION_DENIED"),
	CodeResourceExhausted:  []byte("RESOURCE_EXHAUSTED"),
	CodeFailedPrecondition: []byte("FAILED_PRECONDITION"),
	CodeAborted:            []byte("ABORTED"),
	CodeOutOfRange:         []byte("OUT_OF_RANGE"),
	CodeUnimplemented:      []byte("UNIMPLEMENTED"),
	CodeInternal:           []byte("INTERNAL"),
	CodeUnavailable:        []byte("UNAVAILABLE"),
	CodeDataLoss:           []byte("DATA_LOSS"),
	CodeUnauthenticated:    []byte("UNAUTHENTICATED"),
}

var codeMessages = [...]string{
	CodeOK:                 "The operation completed successfully.",
	CodeCancelled:          "The operation was cancelled.",
	CodeUnknown:            "An unknown system error occurred.",
	CodeInvalidArgument:    "An invalid argument was provided.",
	CodeDeadlineExceeded:   "The deadline expired before the operation could complete.",
	CodeNotFound:           "The requested entity could not be found.",
	CodeAlreadyExists:      "The entity you attempted to create already exists.",
	CodePermissionDenied:   "You do not have permission to execute this operation.",
	CodeResourceExhausted:  "A resource or rate quota has been exhausted.",
	CodeFailedPrecondition: "The operation was rejected because the system is not in a state required for execution.",
	CodeAborted:            "The operation was aborted.",
	CodeOutOfRange:         "The operation was attempted past the valid bounds or index range.",
	CodeUnimplemented:      "This system capability is not implemented or enabled in this service.",
	CodeInternal:           "An internal error occurred.",
	CodeUnavailable:        "The service is temporarily unavailable. Please retry later.",
	CodeDataLoss:           "Unrecoverable data loss or system corruption occurred.",
	CodeUnauthenticated:    "The request lacks valid credentials.",
}

// Parse converts a raw string name (e.g. "NOT_FOUND") into a Code using zero-alloc string conversion.
func Parse(raw string) (Code, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return CodeUnknown, ErrInvalidCode
	}
	for i := 0; i < int(codeMax); i++ {
		if strings.EqualFold(codeNames[i], s) {
			return Code(i), nil
		}
	}
	if n, err := strconv.ParseInt(s, 10, 32); err == nil {
		c := Code(n)
		if c.IsValid() {
			return c, nil
		}
	}
	return CodeUnknown, ErrInvalidCode
}

// ParseBytes parses a byte slice representing a code name or numeric string into a Code.
func ParseBytes(b []byte) (Code, error) {
	b = bytes.TrimSpace(b)
	if len(b) == 0 {
		return CodeUnknown, ErrInvalidCode
	}

	// 1. Check for standard name match (case-insensitive)
	for i := 0; i < int(codeMax); i++ {
		if bytes.EqualFold(codeBytes[i], b) {
			return Code(i), nil
		}
	}

	// 2. Fallback: attempt parsing as a raw integer string (e.g. "5")
	if n, err := strconv.ParseInt(string(b), 10, 32); err == nil {
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
		return 200 // OK
	case CodeInvalidArgument, CodeOutOfRange:
		return 400 // Bad Request
	case CodeUnauthenticated:
		return 401 // Unauthorized
	case CodePermissionDenied:
		return 403 // Forbidden
	case CodeNotFound:
		return 404 // Not Found
	case CodeAlreadyExists, CodeAborted:
		return 409 // Conflict
	case CodeFailedPrecondition:
		return 412 // Precondition Failed
	case CodeResourceExhausted:
		return 429 // Too Many Requests
	case CodeCancelled:
		return 499 // Client Closed Request
	case CodeUnimplemented:
		return 501 // Not Implemented
	case CodeUnavailable:
		return 503 // Service Unavailable
	case CodeDeadlineExceeded:
		return 504 // Gateway Timeout
	default:
		return 500 // Internal Server Error (CodeInternal, CodeUnknown, CodeDataLoss)
	}
}

// --- Encoding Interfaces ---

func (c Code) MarshalText() ([]byte, error) {
	if c.IsValid() {
		return codeBytes[c], nil
	}
	return []byte(c.String()), nil
}

func (c *Code) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		*c = CodeUnknown
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
	if c.IsValid() {
		cb := codeBytes[c]
		b := make([]byte, 0, len(cb)+2)
		b = append(b, '"')
		b = append(b, cb...)
		b = append(b, '"')
		return b, nil
	}
	return json.Marshal(c.String())
}

func (c *Code) UnmarshalJSON(b []byte) error {
	if len(b) == 0 || string(b) == "null" {
		*c = CodeOK
		return nil
	}

	// 1. Handle JSON string representation ("NOT_FOUND")
	if b[0] == '"' && b[len(b)-1] == '"' {
		return c.UnmarshalText(b[1 : len(b)-1])
	}

	// 2. Handle JSON raw numeric representation (5)
	var n int32
	if err := json.Unmarshal(b, &n); err == nil {
		code := Code(n)
		if code.IsValid() {
			*c = code
			return nil
		}
	}

	return ErrInvalidCode
}
