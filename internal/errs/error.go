package errs

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type Error struct {
	Code    Code     `json:"code"`
	Message string   `json:"message"`
	Details []Detail `json:"details,omitempty"`
	Err     error    `json:"-"`
}

func As(err error) *Error {
	if err == nil {
		return nil
	}
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr
	}
	return nil
}

func New(code Code, msg string) *Error {
	if msg == "" {
		msg = code.Message()
	}

	e := &Error{
		Code:    code,
		Message: msg,
	}

	if info, err := NewErrorInfo(code.String(), getDomain(), nil); err == nil {
		e.Detail(info)
	}
	return e
}

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

func (e *Error) Is(target error) bool {
	var t *Error
	if errors.As(target, &t) {
		return e.Code == t.Code
	}
	return false
}

func (e *Error) Error() string {
	if e == nil {
		return "<nil apperr.Error>"
	}
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code.String(), e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code.String(), e.Message)
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func (e *Error) GetDetail(typeURL string) Detail {
	if e == nil {
		return nil
	}
	for _, d := range e.Details {
		if d != nil && d.TypeURL() == typeURL {
			return d
		}
	}
	return nil
}

func (e *Error) Wrap(err error) *Error {
	if e != nil {
		e.Err = err
	}
	return e
}

func (e *Error) Detail(d Detail) *Error {
	if e == nil || d == nil {
		return e
	}
	typeURL := d.TypeURL()

	for i, existing := range e.Details {
		if existing != nil && existing.TypeURL() == typeURL {
			if e.mergeDetail(existing, d) {
				return e
			}
			e.Details[i] = d
			return e
		}
	}
	e.Details = append(e.Details, d)
	return e
}

func (e *Error) Reason(reason string) *Error {
	if e == nil {
		return e
	}
	if !IsValidReason(reason) {
		return e
	}
	if info, ok := e.GetDetail(DetailErrorInfo).(*ErrorInfo); ok && info != nil {
		info.Reason = reason
	}
	return e
}

func (e *Error) Meta(key, value string) *Error {
	if e == nil {
		return e
	}
	if !IsValidMetaKey(key) {
		return e
	}
	if info, ok := e.GetDetail(DetailErrorInfo).(*ErrorInfo); ok && info != nil {
		if info.Metadata == nil {
			info.Metadata = make(map[string]string)
		}
		info.Metadata[key] = value
	}
	return e
}

func (e *Error) Retry(delay time.Duration) *Error {
	if e == nil {
		return nil
	}
	return e.Detail(NewRetryInfo(delay))
}

func (e *Error) Debug(detail string, stack ...string) *Error {
	if e == nil {
		return nil
	}
	if info, err := NewDebugInfo(detail, stack...); err == nil {
		e.Detail(info)
	}
	return e
}

func (e *Error) FieldViolation(field, description, reason string) *Error {
	if e == nil {
		return nil
	}
	if fv, err := NewFieldViolation(field, description, reason); err == nil {
		if br, err := NewBadRequest(*fv); err == nil {
			e.Detail(br)
		}
	}
	return e
}

func (e *Error) Request(requestID, servingData string) *Error {
	if e == nil {
		return nil
	}
	if info, err := NewRequestInfo(requestID, servingData); err == nil {
		e.Detail(info)
	}
	return e
}

func (e *Error) Resource(rType, name, owner, description string) *Error {
	if e == nil {
		return nil
	}
	if info, err := NewResourceInfo(rType, name, owner, description); err == nil {
		e.Detail(info)
	}
	return e
}
func (e *Error) mergeDetail(existing, incoming Detail) bool {
	switch inc := incoming.(type) {

	case *BadRequest:
		if cur, ok := existing.(*BadRequest); ok && cur != nil {
			cur.FieldViolations = append(cur.FieldViolations, inc.FieldViolations...)
			return true
		}

	case *ErrorInfo:
		if cur, ok := existing.(*ErrorInfo); ok && cur != nil {
			if cur.Metadata == nil {
				cur.Metadata = make(map[string]string)
			}
			for k, v := range inc.Metadata {
				cur.Metadata[k] = v
			}
			if inc.Reason != "" && inc.Reason != "REASON_UNSPECIFIED" {
				cur.Reason = inc.Reason
			}
			return true
		}

	case *DebugInfo:
		if cur, ok := existing.(*DebugInfo); ok && cur != nil {
			if inc.Detail != "" {
				cur.Detail = inc.Detail
			}
			cur.StackEntries = append(cur.StackEntries, inc.StackEntries...)
			return true
		}

	case *RetryInfo:
		if cur, ok := existing.(*RetryInfo); ok && cur != nil {
			cur.RetryDelay = inc.RetryDelay
			return true
		}

	case *RequestInfo:
		if cur, ok := existing.(*RequestInfo); ok && cur != nil {
			cur.RequestId = inc.RequestId
			if inc.ServingData != "" {
				cur.ServingData = inc.ServingData
			}
			return true
		}

	case *ResourceInfo:
		if cur, ok := existing.(*ResourceInfo); ok && cur != nil {
			cur.ResourceType = inc.ResourceType
			cur.ResourceName = inc.ResourceName
			if inc.Owner != "" {
				cur.Owner = inc.Owner
			}
			if inc.Description != "" {
				cur.Description = inc.Description
			}
			return true
		}
	}

	return false
}

func (e *Error) UnmarshalJSON(data []byte) error {
	var env struct {
		Code    Code              `json:"code"`
		Message string            `json:"message"`
		Details []json.RawMessage `json:"details,omitempty"`
	}

	if err := json.Unmarshal(data, &env); err != nil {
		return err
	}

	e.Code = env.Code
	e.Message = env.Message

	if len(env.Details) == 0 {
		e.Details = nil
		return nil
	}

	e.Details = make([]Detail, 0, len(env.Details))

	for _, raw := range env.Details {
		var typeExtract struct {
			Type string `json:"@type"`
		}
		if err := json.Unmarshal(raw, &typeExtract); err != nil {
			continue
		}

		var d Detail
		switch typeExtract.Type {
		case DetailErrorInfo:
			d = &ErrorInfo{}
		case DetailRetryInfo:
			d = &RetryInfo{}
		case DetailDebugInfo:
			d = &DebugInfo{}
		case DetailBadRequest:
			d = &BadRequest{}
		case DetailRequestInfo:
			d = &RequestInfo{}
		case DetailResourceInfo:
			d = &ResourceInfo{}
		default:
			e.Details = append(e.Details, &RawDetail{Type: typeExtract.Type, RawData: raw})
			continue
		}

		if err := json.Unmarshal(raw, d); err == nil {
			e.Details = append(e.Details, d)
		} else {
			e.Details = append(e.Details, &RawDetail{Type: typeExtract.Type, RawData: raw})
		}
	}

	return nil
}
