package apperr

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"strings"
)

type Error struct {
	Code    Code     `json:"code"`
	Message string   `json:"message"`
	Details []Detail `json:"details,omitempty"`
	Err     error    `json:"-"`

	// Internal builder state (Used ONLY during New() / Option evaluation)
	detMap map[string]Detail
}

// Error implements the standard error interface. Safe for concurrent access.
func (e *Error) Error() string {
	if e == nil {
		return "<nil apperr.Error>"
	}
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code.String(), e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code.String(), e.Message)
}

// Unwrap implements errors.Unwrap. Safe for concurrent access.
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// GetDetail searches details. Safe for concurrent execution across goroutines.
func (e *Error) GetDetail(typeURL string) Detail {
	if e == nil {
		return nil
	}
	// Post-finalize read path uses immutable slice
	for _, d := range e.Details {
		if d != nil && d.TypeURL() == typeURL {
			return d
		}
	}
	// Pre-finalize construction path uses detMap
	if e.detMap != nil {
		return e.detMap[typeURL]
	}
	return nil
}

func New(code Code, opts ...Option) error {
	e := &Error{
		Code:    code,
		Message: code.Message(),
		detMap:  make(map[string]Detail),
	}

	for _, opt := range opts {
		if opt != nil {
			opt(e)
		}
	}

	if _, exists := e.detMap[DetailErrorInfo]; !exists {
		info, err := NewErrorInfo(code.String(), getDomain(), nil)
		if err == nil {
			e.addDetail(info)
		}
	}

	e.finalize()
	return e
}

// addDetail internal builder method.
func (e *Error) addDetail(d Detail) {
	if e == nil || d == nil {
		return
	}
	val := reflect.ValueOf(d)
	if val.Kind() == reflect.Ptr && val.IsNil() {
		return
	}

	// Post-finalization fallback: append directly to slice
	if e.detMap == nil {
		e.Details = append(e.Details, d)
		return
	}

	typeURL := d.TypeURL()
	existing, exists := e.detMap[typeURL]
	if !exists {
		e.detMap[typeURL] = d
		return
	}

	switch incoming := d.(type) {
	case *BadRequest:
		if current, ok := existing.(*BadRequest); ok && current != nil {
			current.FieldViolations = append(current.FieldViolations, incoming.FieldViolations...)
			return
		}
	case *QuotaFailure:
		if current, ok := existing.(*QuotaFailure); ok && current != nil {
			current.Violations = append(current.Violations, incoming.Violations...)
			return
		}
	case *PreconditionFailure:
		if current, ok := existing.(*PreconditionFailure); ok && current != nil {
			current.Violations = append(current.Violations, incoming.Violations...)
			return
		}
	case *Help:
		if current, ok := existing.(*Help); ok && current != nil {
			current.Links = append(current.Links, incoming.Links...)
			return
		}
	case *ErrorInfo:
		if current, ok := existing.(*ErrorInfo); ok && current != nil {
			if current.Metadata == nil {
				current.Metadata = make(map[string]string)
			}
			for k, v := range incoming.Metadata {
				current.Metadata[k] = v
			}
			if incoming.Reason != "" && incoming.Reason != "REASON_UNSPECIFIED" {
				current.Reason = incoming.Reason
			}
			return
		}
	}

	e.detMap[typeURL] = d
}

func (e *Error) finalize() {
	// 1. Interpolate template placeholders in Message and LocalizedMessage
	if info, ok := e.GetDetail(DetailErrorInfo).(*ErrorInfo); ok && info != nil {
		for k, v := range info.Metadata {
			placeholder := "{" + k + "}"
			e.Message = strings.ReplaceAll(e.Message, placeholder, v)

			if lm, ok := e.GetDetail(DetailLocalizedMessage).(*LocalizedMessage); ok && lm != nil {
				lm.Message = strings.ReplaceAll(lm.Message, placeholder, v)
			}
		}
	}

	// 2. Ensure Help URLs contain error reason anchor fragment
	if info, ok := e.GetDetail(DetailErrorInfo).(*ErrorInfo); ok && info != nil && info.Reason != "" {
		if h, ok := e.GetDetail(DetailHelp).(*Help); ok && h != nil {
			for i := range h.Links {
				u, err := url.Parse(h.Links[i].URL)
				if err == nil && u.Fragment == "" {
					u.Fragment = info.Reason
					h.Links[i].URL = u.String()
				}
			}
		}
	}

	// 3. Convert builder map to slice for clean JSON output
	e.Details = make([]Detail, 0, len(e.detMap))
	for _, d := range e.detMap {
		e.Details = append(e.Details, d)
	}

	// 4. Wipe internal builder state to make object immutable
	e.detMap = nil
}

func IsCode(err error, code Code) bool {
	var appErr *Error
	return errors.As(err, &appErr) && appErr != nil && appErr.Code == code
}

// UnmarshalJSON polymorphic unmarshaler for Details interface slice.
func (e *Error) UnmarshalJSON(data []byte) error {
	type Alias Error
	aux := &struct {
		RawDetails []json.RawMessage `json:"details,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(e),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	e.Details = make([]Detail, 0, len(aux.RawDetails))
	for _, raw := range aux.RawDetails {
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
		case DetailQuotaFailure:
			d = &QuotaFailure{}
		case DetailPreconditionFailure:
			d = &PreconditionFailure{}
		case DetailBadRequest:
			d = &BadRequest{}
		case DetailRequestInfo:
			d = &RequestInfo{}
		case DetailResourceInfo:
			d = &ResourceInfo{}
		case DetailHelp:
			d = &Help{}
		case DetailLocalizedMessage:
			d = &LocalizedMessage{}
		default:
			continue
		}

		if err := json.Unmarshal(raw, d); err == nil {
			e.Details = append(e.Details, d)
		}
	}

	return nil
}
