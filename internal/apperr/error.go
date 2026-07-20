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

	// detMap is internal construction state. Wiped during finalize().
	detMap map[string]Detail
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

// GetDetail fetches a detail by its type URL.
func (e *Error) GetDetail(typeURL string) Detail {
	if e == nil {
		return nil
	}
	for _, d := range e.Details {
		if d != nil && d.TypeURL() == typeURL {
			return d
		}
	}
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

	if info, err := NewErrorInfo(code.String(), getDomain(), nil); err == nil {
		e.addDetail(info)
	}

	for _, opt := range opts {
		if opt != nil {
			opt(e)
		}
	}

	e.finalize()
	return e
}

func (e *Error) addDetail(d Detail) {
	if e == nil || d == nil {
		return
	}
	val := reflect.ValueOf(d)
	if val.Kind() == reflect.Ptr && val.IsNil() {
		return
	}

	typeURL := d.TypeURL()

	// 1. Post-finalize state (detMap is nil)
	if e.detMap == nil {
		for i, existing := range e.Details {
			if existing != nil && existing.TypeURL() == typeURL {
				if e.mergeDetail(existing, d) {
					return
				}
				e.Details[i] = d
				return
			}
		}
		e.Details = append(e.Details, d)
		return
	}

	// 2. Pre-finalize state (detMap active)
	if existing, exists := e.detMap[typeURL]; exists {
		if e.mergeDetail(existing, d) {
			return
		}
	}

	e.detMap[typeURL] = d
}

// Helper to prevent duplicate merge logic between detMap and Details slice
func (e *Error) mergeDetail(existing, incoming Detail) bool {
	switch inc := incoming.(type) {
	case *BadRequest:
		if cur, ok := existing.(*BadRequest); ok && cur != nil {
			cur.FieldViolations = append(cur.FieldViolations, inc.FieldViolations...)
			return true
		}
	case *QuotaFailure:
		if cur, ok := existing.(*QuotaFailure); ok && cur != nil {
			cur.Violations = append(cur.Violations, inc.Violations...)
			return true
		}
	case *PreconditionFailure:
		if cur, ok := existing.(*PreconditionFailure); ok && cur != nil {
			cur.Violations = append(cur.Violations, inc.Violations...)
			return true
		}
	case *Help:
		if cur, ok := existing.(*Help); ok && cur != nil {
			cur.Links = append(cur.Links, inc.Links...)
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
	}
	return false
}

func (e *Error) finalize() {
	// 1. Interpolate placeholders using metadata
	if info, ok := e.GetDetail(DetailErrorInfo).(*ErrorInfo); ok && info != nil {
		for k, v := range info.Metadata {
			placeholder := "{" + k + "}"
			e.Message = strings.ReplaceAll(e.Message, placeholder, v)

			if lm, ok := e.GetDetail(DetailLocalizedMessage).(*LocalizedMessage); ok && lm != nil {
				lm.Message = strings.ReplaceAll(lm.Message, placeholder, v)
			}
		}
	}

	// 2. Attach reason fragments to Help URLs
	if info, ok := e.GetDetail(DetailErrorInfo).(*ErrorInfo); ok && info != nil && info.Reason != "" {
		if h, ok := e.GetDetail(DetailHelp).(*Help); ok && h != nil {
			for i := range h.Links {
				u, err := url.Parse(h.Links[i].URL)
				if err == nil && u.Fragment == "" {
					u.Fragment = url.PathEscape(info.Reason)
					h.Links[i].URL = u.String()
				}
			}
		}
	}

	// 3. Move map items to immutable slice in deterministic order
	// (Standard order: ErrorInfo first, then rest)
	e.Details = make([]Detail, 0, len(e.detMap))

	// Always put ErrorInfo first if present for clean JSON inspection
	if errInfo, ok := e.detMap[DetailErrorInfo]; ok {
		e.Details = append(e.Details, errInfo)
	}

	for typeURL, d := range e.detMap {
		if typeURL != DetailErrorInfo {
			e.Details = append(e.Details, d)
		}
	}

	e.detMap = nil
}

func IsCode(err error, code Code) bool {
	var appErr *Error
	return errors.As(err, &appErr) && appErr != nil && appErr.Code == code
}

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
			// Custom / Unknown detail types preservation
			d = &RawDetail{Type: typeExtract.Type, RawData: raw}
			e.Details = append(e.Details, d)
			continue
		}

		if err := json.Unmarshal(raw, d); err == nil {
			e.Details = append(e.Details, d)
		}
	}

	return nil
}
