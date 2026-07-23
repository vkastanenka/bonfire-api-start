package errs

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"
)

// https://github.com/googleapis/googleapis/blob/master/google/rpc/error_details.proto

const (
	DetailErrorInfo    = "ErrorInfo"
	DetailRetryInfo    = "RetryInfo"
	DetailBadRequest   = "BadRequest"
	DetailDebugInfo    = "DebugInfo"
	DetailRequestInfo  = "RequestInfo"
	DetailResourceInfo = "ResourceInfo"
)

type Detail interface {
	TypeURL() string
}

type RawDetail struct {
	Type    string          `json:"@type"`
	RawData json.RawMessage `json:"-"`
}

func (r *RawDetail) TypeURL() string { return r.Type }

type ErrorInfo struct {
	Type     string            `json:"@type"`
	Reason   string            `json:"reason"`
	Domain   string            `json:"domain"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

var reasonRegex = regexp.MustCompile(`^[A-Z][A-Z0-9_]+[A-Z0-9]$`)

func IsValidReason(reason string) bool {
	if len(reason) == 0 || len(reason) > 63 {
		return false
	}
	return reasonRegex.MatchString(reason)
}

var metaKeyRegex = regexp.MustCompile(`^[a-z][a-zA-Z0-9-_]+$`)

func IsValidMetaKey(key string) bool {
	if len(key) == 0 || len(key) > 64 {
		return false
	}
	return metaKeyRegex.MatchString(key)
}

func NewErrorInfo(reason, domain string, metadata map[string]string) (*ErrorInfo, error) {
	if domain == "" {
		return nil, errors.New("apperr: ErrorInfo domain is required")
	}
	if reason != "" && !IsValidReason(reason) {
		return nil, fmt.Errorf("apperr: invalid ErrorInfo reason %q (must be UPPER_SNAKE_CASE, <=63 chars)", reason)
	}

	for key := range metadata {
		if !IsValidMetaKey(key) {
			return nil, fmt.Errorf("apperr: invalid ErrorInfo metadata key %q (must match [a-z][a-zA-Z0-9-_]+ and be <=64 chars)", key)
		}
	}

	return &ErrorInfo{
		Type:     DetailErrorInfo,
		Reason:   reason,
		Domain:   domain,
		Metadata: metadata,
	}, nil
}

func (e *ErrorInfo) TypeURL() string { return DetailErrorInfo }

type RetryInfo struct {
	Type       string        `json:"@type"`
	RetryDelay time.Duration `json:"-"`
}

func NewRetryInfo(delay time.Duration) *RetryInfo {
	return &RetryInfo{Type: DetailRetryInfo, RetryDelay: delay}
}

func (r *RetryInfo) TypeURL() string { return DetailRetryInfo }

func (r *RetryInfo) MarshalJSON() ([]byte, error) {
	if r == nil {
		return json.Marshal(nil)
	}
	return json.Marshal(&struct {
		Type       string `json:"@type"`
		RetryDelay string `json:"retryDelay"`
	}{
		Type:       DetailRetryInfo,
		RetryDelay: fmt.Sprintf("%.9fs", r.RetryDelay.Seconds()),
	})
}

func (r *RetryInfo) UnmarshalJSON(b []byte) error {
	if r == nil {
		return errors.New("apperr: UnmarshalJSON on nil RetryInfo pointer")
	}

	var raw struct {
		Type     string `json:"@type"`
		DelayRaw string `json:"retryDelay"`
	}

	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}

	r.Type = DetailRetryInfo
	if raw.DelayRaw != "" {
		d, err := time.ParseDuration(raw.DelayRaw)
		if err != nil {
			return fmt.Errorf("apperr: invalid retryDelay duration %q: %w", raw.DelayRaw, err)
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
		return nil, errors.New("apperr: DebugInfo detail is required")
	}
	return &DebugInfo{
		Type:         DetailDebugInfo,
		Detail:       detail,
		StackEntries: stackEntries,
	}, nil
}

func (d *DebugInfo) TypeURL() string { return DetailDebugInfo }

type FieldViolation struct {
	Field       string `json:"field"`
	Description string `json:"description"`
	Reason      string `json:"reason"`
}

func NewFieldViolation(field, description, reason string) (*FieldViolation, error) {
	if field == "" || description == "" || reason == "" {
		return nil, errors.New("apperr: FieldViolation field, description, and reason are required")
	}
	return &FieldViolation{Field: field, Description: description, Reason: reason}, nil
}

type BadRequest struct {
	Type            string           `json:"@type"`
	FieldViolations []FieldViolation `json:"fieldViolations,omitempty"`
}

func NewBadRequest(violations ...FieldViolation) (*BadRequest, error) {
	if len(violations) == 0 {
		return nil, errors.New("apperr: BadRequest requires at least one field violation")
	}
	return &BadRequest{Type: DetailBadRequest, FieldViolations: violations}, nil
}

func (b *BadRequest) TypeURL() string { return DetailBadRequest }

type RequestInfo struct {
	Type        string `json:"@type"`
	RequestId   string `json:"requestId"`
	ServingData string `json:"servingData,omitempty"`
}

func NewRequestInfo(requestID, servingData string) (*RequestInfo, error) {
	if requestID == "" {
		return nil, errors.New("apperr: RequestInfo requestID is required")
	}
	return &RequestInfo{Type: DetailRequestInfo, RequestId: requestID, ServingData: servingData}, nil
}

func (r *RequestInfo) TypeURL() string { return DetailRequestInfo }

type ResourceInfo struct {
	Type         string `json:"@type"`
	ResourceType string `json:"resourceType"`
	ResourceName string `json:"resourceName"`
	Owner        string `json:"owner,omitempty"`
	Description  string `json:"description,omitempty"`
}

func NewResourceInfo(rType, rName, owner, desc string) (*ResourceInfo, error) {
	if rType == "" || rName == "" {
		return nil, errors.New("apperr: ResourceInfo resourceType and resourceName are required")
	}
	return &ResourceInfo{Type: DetailResourceInfo, ResourceType: rType, ResourceName: rName, Owner: owner, Description: desc}, nil
}

func (r *ResourceInfo) TypeURL() string { return DetailResourceInfo }
