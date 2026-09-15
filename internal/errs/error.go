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
	Details []Detail `json:"details"`
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

	info, err := NewErrorInfo(code.Name(), getDomain(), nil)
	if err == nil {
		e.AddDetail(info)
	}
	return e
}

func IsCode(err error, code Code) bool {
	var appErr *Error
	if errors.As(err, &appErr) {
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
		return fmt.Sprintf("[%s] %s: %v", e.Code.Name(), e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code.Name(), e.Message)
}

func (e *Error) GetDetail(typeURL string) Detail {
	if e == nil {
		return nil
	}

	targetClean := cleanDetailTypeURL(typeURL)

	for _, d := range e.Details {
		if d == nil {
			continue
		}
		if d.TypeURL() == typeURL || cleanDetailTypeURL(d.TypeURL()) == targetClean {
			return d
		}
	}
	return nil
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func (e *Error) Wrap(err error) *Error {
	if e != nil {
		e.Err = err
	}
	return e
}

func (e *Error) AddDetail(d Detail) *Error {
	if e == nil || d == nil {
		return e
	}
	targetClean := cleanDetailTypeURL(d.TypeURL())

	for i, existing := range e.Details {
		if existing != nil && cleanDetailTypeURL(existing.TypeURL()) == targetClean {
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

func (e *Error) ErrorInfoReason(reason string) *Error {
	if e == nil || !isValidErrorInfoReason(reason) {
		return e
	}
	info, ok := e.GetDetail(DetailErrorInfo).(*ErrorInfo)
	if !ok || info == nil {
		var err error
		info, err = NewErrorInfo(reason, getDomain(), nil)
		if err != nil {
			return e
		}
		return e.AddDetail(info)
	}
	info.Reason = reason
	return e
}

func (e *Error) ErrorInfoMeta(key, value string) *Error {
	if e == nil {
		return e
	}
	if !isValidErrorInfoMetaKey(key) {
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

func (e *Error) RetryInfo(delay time.Duration) *Error {
	if e == nil {
		return e
	}
	return e.AddDetail(NewRetryInfo(delay))
}

func (e *Error) DebugInfo(detail string, stack ...string) *Error {
	if e == nil {
		return e
	}
	if info, err := NewDebugInfo(detail, stack...); err == nil {
		e.AddDetail(info)
	}
	return e
}

func (e *Error) FieldViolation(field, description, reason string) *Error {
	if e == nil {
		return e
	}

	fv, err := NewFieldViolation(field, description, reason)
	if err != nil {
		return e
	}

	if br, ok := e.GetDetail(DetailBadRequest).(*BadRequest); ok && br != nil {
		br.FieldViolations = append(br.FieldViolations, *fv)
		return e
	}

	if br, err := NewBadRequest(*fv); err == nil {
		e.AddDetail(br)
	}

	return e
}

func (e *Error) RequestInfo(requestID, servingData string) *Error {
	if e == nil {
		return e
	}
	if info, err := NewRequestInfo(requestID, servingData); err == nil {
		e.AddDetail(info)
	}
	return e
}

func (e *Error) ResourceInfo(rType, name, owner, description string) *Error {
	if e == nil {
		return e
	}
	if info, err := NewResourceInfo(rType, name, owner, description); err == nil {
		e.AddDetail(info)
	}
	return e
}

type errorJSON struct {
	Code    Code     `json:"code"`
	Message string   `json:"message"`
	Details []Detail `json:"details,omitempty"`
}

func (e *Error) MarshalJSON() ([]byte, error) {
	return json.Marshal(errorJSON{
		Code:    e.Code,
		Message: e.Message,
		Details: e.Details,
	})
}

func (e *Error) UnmarshalJSON(data []byte) error {
	var env struct {
		Code    Code              `json:"code"`
		Message string            `json:"message"`
		Details []json.RawMessage `json:"details,omitempty"`
	}

	if err := json.Unmarshal(data, &env); err != nil {
		return fmt.Errorf("unmarshal error envelope: %w", err)
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
		if err := json.Unmarshal(raw, &typeExtract); err != nil || typeExtract.Type == "" {
			e.Details = append(e.Details, &RawDetail{RawData: raw})
			continue
		}

		factory, ok := detailRegistry[typeExtract.Type]
		if !ok {
			factory, ok = detailRegistry[cleanDetailTypeURL(typeExtract.Type)]
		}

		if !ok {
			e.Details = append(e.Details, &RawDetail{Type: typeExtract.Type, RawData: raw})
			continue
		}

		d := factory()
		if err := json.Unmarshal(raw, d); err != nil {
			e.Details = append(e.Details, &RawDetail{Type: typeExtract.Type, RawData: raw})
			continue
		}

		e.Details = append(e.Details, d)
	}

	return nil
}

func (e *Error) mergeDetail(existing, incoming Detail) bool {
	switch cur := existing.(type) {
	case *BadRequest:
		if inc, ok := incoming.(*BadRequest); ok && inc != nil {
			cur.FieldViolations = append(cur.FieldViolations, inc.FieldViolations...)
			return true
		}
	case *ErrorInfo:
		if inc, ok := incoming.(*ErrorInfo); ok && inc != nil {
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
		if inc, ok := incoming.(*DebugInfo); ok && inc != nil {
			if inc.Detail != "" {
				cur.Detail = inc.Detail
			}
			cur.StackEntries = append(cur.StackEntries, inc.StackEntries...)
			return true
		}
	case *RetryInfo:
		if inc, ok := incoming.(*RetryInfo); ok && inc != nil {
			cur.RetryDelay = inc.RetryDelay
			return true
		}
	case *RequestInfo:
		if inc, ok := incoming.(*RequestInfo); ok && inc != nil {
			cur.RequestId = inc.RequestId
			if inc.ServingData != "" {
				cur.ServingData = inc.ServingData
			}
			return true
		}
	case *ResourceInfo:
		if inc, ok := incoming.(*ResourceInfo); ok && inc != nil {
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
