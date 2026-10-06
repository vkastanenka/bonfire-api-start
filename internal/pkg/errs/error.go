package errs

import (
	"errors"
	"time"
)

type Error struct {
	Code    Code   `json:"code"`
	Message string `json:"message"`
	Details []any  `json:"details"`
	Err     error  `json:"-"`
}

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

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func (e *Error) Is(target error) bool {
	if e == nil {
		return target == nil
	}
	var t *Error
	if errors.As(target, &t) && t != nil {
		return e.Code == t.Code
	}
	return false
}

func (e *Error) Wrap(err error) *Error {
	if e != nil {
		e.Err = err
	}
	return e
}

func IsCode(err error, code Code) bool {
	if appErr := From(err); appErr != nil {
		return appErr.Code == code
	}
	return false
}

func IsNotFound(err error) bool { return IsCode(err, CodeNotFound) }

func Cancelled(msg string) *Error          { return New(CodeCancelled, msg) }
func InvalidArgument(msg string) *Error    { return New(CodeInvalidArgument, msg) }
func DeadlineExceeded(msg string) *Error   { return New(CodeDeadlineExceeded, msg) }
func NotFound(msg string) *Error           { return New(CodeNotFound, msg) }
func AlreadyExists(msg string) *Error      { return New(CodeAlreadyExists, msg) }
func PermissionDenied(msg string) *Error   { return New(CodePermissionDenied, msg) }
func ResourceExhausted(msg string) *Error  { return New(CodeResourceExhausted, msg) }
func FailedPrecondition(msg string) *Error { return New(CodeFailedPrecondition, msg) }
func Aborted(msg string) *Error            { return New(CodeAborted, msg) }
func OutOfRange(msg string) *Error         { return New(CodeOutOfRange, msg) }
func Unimplemented(msg string) *Error      { return New(CodeUnimplemented, msg) }
func Internal(msg string) *Error           { return New(CodeInternal, msg) }
func Unavailable(msg string) *Error        { return New(CodeUnavailable, msg) }
func DataLoss(msg string) *Error           { return New(CodeDataLoss, msg) }
func Unauthenticated(msg string) *Error    { return New(CodeUnauthenticated, msg) }

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
func (e *Error) Retry(delay time.Duration) *Error {
	if e == nil {
		return e
	}
	return e.Detail(NewRetryInfo(delay))
}

func (e *Error) DebugInfo(detail string, stack ...string) *Error {
	if e == nil {
		return e
	}
	return e.Detail(NewDebugInfo(detail, stack...))
}

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

func (e *Error) Request(requestID, servingData string) *Error {
	if e == nil {
		return e
	}
	return e.Detail(NewRequestInfo(requestID, servingData))
}

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
