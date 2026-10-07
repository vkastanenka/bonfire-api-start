package errs

import (
	"errors"
	"time"
)

// Error represents a structured, standard-compliant error implementation holding a gRPC-style
// status code, user-facing message, optional wrapped causal error, and metadata payload list.
type Error struct {
	Code    Code   `json:"code"`
	Message string `json:"message"`
	Details []any  `json:"details,omitempty"`
	Err     error  `json:"-"`
}

// New constructs a new *Error with the specified code and message. Automatically initializes
// and appends a default ErrorInfo payload containing the code name as reason. Fallbacks to
// default code message if msg is empty.
func New(code Code, msg string) *Error {
	if msg == "" {
		msg = code.Message()
	}

	e := &Error{
		Code:    code,
		Message: msg,
	}

	info := NewErrorInfo(code.Name(), getDomain(), nil)
	e.Detail(info)

	return e
}

// From extracts *Error from error chain using errors.As. Returns nil if target error
// does not contain *Error.
func From(err error) *Error {
	if err == nil {
		return nil
	}
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr
	}
	return nil
}

// Error formats error message into readable string including code name and underlying
// wrapped error if present.
func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}

	var msg string
	if e.Code != 0 {
		msg = e.Code.Name() + ": " + e.Message
	} else {
		msg = e.Message
	}

	if e.Err != nil {
		return msg + ": " + e.Err.Error()
	}

	return msg
}

// Unwrap returns wrapped inner error Err for errors.Unwrap compatibility.
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// Is compares whether target error shares the same Code.
func (e *Error) Is(target error) bool {
	if appErr := From(target); appErr != nil {
		return e.Code == appErr.Code
	}
	return false
}

// Wrap sets underlying wrapped Err and returns mutated *Error.
func (e *Error) Wrap(err error) *Error {
	if e != nil {
		e.Err = err
	}
	return e
}

// IsCode checks whether any error in chain resolves to *Error with matching code.
func IsCode(err error, code Code) bool {
	if appErr := From(err); appErr != nil {
		return appErr.Code == code
	}
	return false
}

// IsNotFound is a convenience helper for IsCode(err, CodeNotFound).
func IsNotFound(err error) bool { return IsCode(err, CodeNotFound) }

// Code-specific convenience constructors

// Cancelled constructs a CodeCancelled error with custom message.
func Cancelled(msg string) *Error { return New(CodeCancelled, msg) }

// InvalidArgument constructs a CodeInvalidArgument error with custom message.
func InvalidArgument(msg string) *Error { return New(CodeInvalidArgument, msg) }

// DeadlineExceeded constructs a CodeDeadlineExceeded error with custom message.
func DeadlineExceeded(msg string) *Error { return New(CodeDeadlineExceeded, msg) }

// NotFound constructs a CodeNotFound error with custom message.
func NotFound(msg string) *Error { return New(CodeNotFound, msg) }

// AlreadyExists constructs a CodeAlreadyExists error with custom message.
func AlreadyExists(msg string) *Error { return New(CodeAlreadyExists, msg) }

// PermissionDenied constructs a CodePermissionDenied error with custom message.
func PermissionDenied(msg string) *Error { return New(CodePermissionDenied, msg) }

// ResourceExhausted constructs a CodeResourceExhausted error with custom message.
func ResourceExhausted(msg string) *Error { return New(CodeResourceExhausted, msg) }

// FailedPrecondition constructs a CodeFailedPrecondition error with custom message.
func FailedPrecondition(msg string) *Error { return New(CodeFailedPrecondition, msg) }

// Aborted constructs a CodeAborted error with custom message.
func Aborted(msg string) *Error { return New(CodeAborted, msg) }

// OutOfRange constructs a CodeOutOfRange error with custom message.
func OutOfRange(msg string) *Error { return New(CodeOutOfRange, msg) }

// Unimplemented constructs a CodeUnimplemented error with custom message.
func Unimplemented(msg string) *Error { return New(CodeUnimplemented, msg) }

// Internal constructs a CodeInternal error with custom message.
func Internal(msg string) *Error { return New(CodeInternal, msg) }

// Unavailable constructs a CodeUnavailable error with custom message.
func Unavailable(msg string) *Error { return New(CodeUnavailable, msg) }

// DataLoss constructs a CodeDataLoss error with custom message.
func DataLoss(msg string) *Error { return New(CodeDataLoss, msg) }

// Unauthenticated constructs a CodeUnauthenticated error with custom message.
func Unauthenticated(msg string) *Error { return New(CodeUnauthenticated, msg) }

// Fluent Builder Interface Methods

// Detail appends or updates detail payload in e.Details matching type URL.
func (e *Error) Detail(d any) *Error {
	if e == nil || d == nil {
		return e
	}

	incURL := detailURL(d)
	if incURL == "" {
		return e
	}

	for i, existing := range e.Details {
		if detailURL(existing) == incURL {
			e.Details[i] = d
			return e
		}
	}

	e.Details = append(e.Details, d)
	return e
}

// Reason updates reason in existing ErrorInfo payload or appends new ErrorInfo with specified reason.
func (e *Error) Reason(reason string) *Error {
	if e == nil || !isValidErrorInfoReason(reason) {
		return e
	}

	if info, ok := getDetail[*ErrorInfo](e); ok && info != nil {
		info.Reason = reason
		return e
	}

	return e.Detail(NewErrorInfo(reason, getDomain(), nil))
}

// Meta sets metadata key-value pair in ErrorInfo payload.
func (e *Error) Meta(key, value string) *Error {
	if e == nil || !isValidErrorInfoMetaKey(key) {
		return e
	}

	info, ok := getDetail[*ErrorInfo](e)
	if !ok || info == nil {
		info = NewErrorInfo(e.Code.Name(), getDomain(), map[string]string{key: value})
		return e.Detail(info)
	}

	if info.Metadata == nil {
		info.Metadata = make(map[string]string)
	}

	info.Metadata[key] = value
	return e
}

// Retry appends RetryInfo detail.
func (e *Error) Retry(delay time.Duration) *Error {
	if e == nil {
		return e
	}
	return e.Detail(NewRetryInfo(delay))
}

// DebugInfo appends DebugInfo detail.
func (e *Error) DebugInfo(detail string, stack ...string) *Error {
	if e == nil {
		return e
	}
	return e.Detail(NewDebugInfo(detail, stack...))
}

// FieldViolation appends FieldViolation to BadRequest detail, creating BadRequest payload if absent.
func (e *Error) FieldViolation(field, description, reason string) *Error {
	if e == nil {
		return e
	}

	fv := NewFieldViolation(field, description, reason)
	if fv == nil {
		return e
	}

	if br, ok := getDetail[*BadRequest](e); ok && br != nil {
		br.FieldViolations = append(br.FieldViolations, *fv)
		return e
	}

	return e.Detail(NewBadRequest(*fv))
}

// Request appends RequestInfo detail.
func (e *Error) Request(requestID, servingData string) *Error {
	if e == nil {
		return e
	}
	return e.Detail(NewRequestInfo(requestID, servingData))
}

// Resource appends ResourceInfo detail.
func (e *Error) Resource(rType, name, owner, description string) *Error {
	if e == nil {
		return e
	}
	return e.Detail(NewResourceInfo(rType, name, owner, description))
}

func detailURL(d any) string {
	if v, ok := d.(Detail); ok && v != nil {
		return v.TypeURL()
	}
	return ""
}

func getDetail[T any](e *Error) (T, bool) {
	var zero T
	if e == nil {
		return zero, false
	}

	for _, d := range e.Details {
		if typed, ok := d.(T); ok {
			return typed, true
		}
	}

	return zero, false
}
