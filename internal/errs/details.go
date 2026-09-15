package errs

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// https://github.com/googleapis/googleapis/blob/master/google/rpc/error_details.proto

const (
	detailTypeURLPrefix = "type.googleapis.com/google.rpc."

	DetailErrorInfo    = detailTypeURLPrefix + "ErrorInfo"
	DetailRetryInfo    = detailTypeURLPrefix + "RetryInfo"
	DetailDebugInfo    = detailTypeURLPrefix + "DebugInfo"
	DetailBadRequest   = detailTypeURLPrefix + "BadRequest"
	DetailRequestInfo  = detailTypeURLPrefix + "RequestInfo"
	DetailResourceInfo = detailTypeURLPrefix + "ResourceInfo"
)

var detailRegistry = map[string]func() Detail{
	DetailErrorInfo:                        func() Detail { return &ErrorInfo{} },
	cleanDetailTypeURL(DetailErrorInfo):    func() Detail { return &ErrorInfo{} },
	DetailRetryInfo:                        func() Detail { return &RetryInfo{} },
	cleanDetailTypeURL(DetailRetryInfo):    func() Detail { return &RetryInfo{} },
	DetailDebugInfo:                        func() Detail { return &DebugInfo{} },
	cleanDetailTypeURL(DetailDebugInfo):    func() Detail { return &DebugInfo{} },
	DetailBadRequest:                       func() Detail { return &BadRequest{} },
	cleanDetailTypeURL(DetailBadRequest):   func() Detail { return &BadRequest{} },
	DetailRequestInfo:                      func() Detail { return &RequestInfo{} },
	cleanDetailTypeURL(DetailRequestInfo):  func() Detail { return &RequestInfo{} },
	DetailResourceInfo:                     func() Detail { return &ResourceInfo{} },
	cleanDetailTypeURL(DetailResourceInfo): func() Detail { return &ResourceInfo{} },
}

type Detail interface {
	TypeURL() string
}

type RawDetail struct {
	Type    string          `json:"@type"`
	RawData json.RawMessage `json:"-"`
}

func (r *RawDetail) TypeURL() string { return r.Type }

func (r *RawDetail) MarshalJSON() ([]byte, error) {
	if len(r.RawData) > 0 {
		return r.RawData, nil
	}
	return json.Marshal(struct {
		Type string `json:"@type"`
	}{Type: r.Type})
}

func (r *RawDetail) UnmarshalJSON(data []byte) error {
	r.RawData = append(r.RawData[:0], data...)

	var typeExtract struct {
		Type string `json:"@type"`
	}
	if err := json.Unmarshal(data, &typeExtract); err != nil {
		return err
	}

	r.Type = typeExtract.Type
	return nil
}

type ErrorInfo struct {
	Type     string            `json:"@type"`
	Reason   string            `json:"reason"`
	Domain   string            `json:"domain"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

var (
	errorInfoReasonMaxLength  = 63
	errorInfoMetaKeyMaxLength = 64
	errorInfoReasonRegex      = regexp.MustCompile(`^[A-Z][A-Z0-9_]+[A-Z0-9]$`)
	errorInfoMetaKeyRegex     = regexp.MustCompile(`^[a-z][a-zA-Z0-9-_]+$`)
)

func NewErrorInfo(reason, domain string, metadata map[string]string) (*ErrorInfo, error) {
	if domain == "" {
		return nil, errors.New("error info domain is required")
	}

	if reason != "" && !isValidErrorInfoReason(reason) {
		return nil, fmt.Errorf("invalid ErrorInfo reason %q (must be UPPER_SNAKE_CASE, <=63 chars)", reason)
	}

	for key := range metadata {
		if !isValidErrorInfoMetaKey(key) {
			return nil, fmt.Errorf("invalid ErrorInfo metadata key %q (must match [a-z][a-zA-Z0-9-_]+ and be <=64 chars)", key)
		}
	}

	return &ErrorInfo{
		Type:     DetailErrorInfo,
		Reason:   reason,
		Domain:   domain,
		Metadata: metadata,
	}, nil
}

func (e *ErrorInfo) TypeURL() string { return e.Type }

func isValidErrorInfoReason(reason string) bool {
	if len(reason) == 0 || len(reason) > errorInfoReasonMaxLength {
		return false
	}
	return errorInfoReasonRegex.MatchString(reason)
}

func isValidErrorInfoMetaKey(key string) bool {
	if len(key) == 0 || len(key) > errorInfoMetaKeyMaxLength {
		return false
	}
	return errorInfoMetaKeyRegex.MatchString(key)
}

type RetryInfo struct {
	Type       string        `json:"@type"`
	RetryDelay time.Duration `json:"-"`
}

type retryInfoJSON struct {
	Type       string `json:"@type"`
	RetryDelay string `json:"retryDelay,omitempty"`
}

func NewRetryInfo(delay time.Duration) *RetryInfo {
	return &RetryInfo{Type: DetailRetryInfo, RetryDelay: delay}
}

func (r *RetryInfo) TypeURL() string { return r.Type }

func (r *RetryInfo) MarshalJSON() ([]byte, error) {
	var delayStr string
	if r.RetryDelay > 0 {
		delayStr = fmt.Sprintf("%.9fs", r.RetryDelay.Seconds())
	}

	return json.Marshal(retryInfoJSON{
		Type:       r.Type,
		RetryDelay: delayStr,
	})
}

func (r *RetryInfo) UnmarshalJSON(b []byte) error {
	var aux retryInfoJSON
	if err := json.Unmarshal(b, &aux); err != nil {
		return err
	}

	if aux.Type != "" {
		r.Type = aux.Type
	} else {
		r.Type = DetailRetryInfo
	}

	if aux.RetryDelay != "" {
		d, err := time.ParseDuration(aux.RetryDelay)
		if err != nil {
			return fmt.Errorf("invalid retryDelay duration %q: %w", aux.RetryDelay, err)
		}
		r.RetryDelay = d
	}

	return nil
}

type DebugInfo struct {
	Type         string   `json:"@type"`
	Detail       string   `json:"detail,omitempty"`
	StackEntries []string `json:"stackEntries,omitempty"`
}

func NewDebugInfo(detail string, stackEntries ...string) (*DebugInfo, error) {
	if detail == "" {
		return nil, errors.New("debug info detail is required")
	}
	return &DebugInfo{
		Type:         DetailDebugInfo,
		Detail:       detail,
		StackEntries: stackEntries,
	}, nil
}

func (d *DebugInfo) TypeURL() string { return d.Type }

type FieldViolation struct {
	Field       string `json:"field"`
	Description string `json:"description"`
	Reason      string `json:"reason,omitempty"`
}

func NewFieldViolation(field, description, reason string) (*FieldViolation, error) {
	if field == "" || description == "" {
		return nil, errors.New("field violation field and description are required")
	}
	return &FieldViolation{Field: field, Description: description, Reason: reason}, nil
}

type BadRequest struct {
	Type            string           `json:"@type"`
	FieldViolations []FieldViolation `json:"fieldViolations,omitempty"`
}

func NewBadRequest(violations ...FieldViolation) (*BadRequest, error) {
	if len(violations) == 0 {
		return nil, errors.New("bad request requires at least one field violation")
	}
	return &BadRequest{Type: DetailBadRequest, FieldViolations: violations}, nil
}

func (b *BadRequest) TypeURL() string { return b.Type }

type RequestInfo struct {
	Type        string `json:"@type"`
	RequestId   string `json:"requestId"`
	ServingData string `json:"servingData,omitempty"`
}

func NewRequestInfo(requestID, servingData string) (*RequestInfo, error) {
	if requestID == "" {
		return nil, errors.New("request info requestID is required")
	}
	return &RequestInfo{Type: DetailRequestInfo, RequestId: requestID, ServingData: servingData}, nil
}

func (r *RequestInfo) TypeURL() string { return r.Type }

type ResourceInfo struct {
	Type         string `json:"@type"`
	ResourceType string `json:"resourceType"`
	ResourceName string `json:"resourceName"`
	Owner        string `json:"owner,omitempty"`
	Description  string `json:"description,omitempty"`
}

func NewResourceInfo(rType, rName, owner, desc string) (*ResourceInfo, error) {
	if rType == "" || rName == "" {
		return nil, errors.New("resource info resourceType and resourceName are required")
	}
	return &ResourceInfo{Type: DetailResourceInfo, ResourceType: rType, ResourceName: rName, Owner: owner, Description: desc}, nil
}

func (r *ResourceInfo) TypeURL() string { return r.Type }

func cleanDetailTypeURL(rawType string) string {
	if idx := strings.LastIndex(rawType, "/"); idx != -1 {
		rawType = rawType[idx+1:]
	}
	return strings.TrimPrefix(rawType, "google.rpc.")
}
