package errs

import (
	"regexp"
	"time"
)

const (
	detailTypeURLPrefix = "type.googleapis.com/google.rpc."

	DetailErrorInfo    = detailTypeURLPrefix + "ErrorInfo"
	DetailRetryInfo    = detailTypeURLPrefix + "RetryInfo"
	DetailDebugInfo    = detailTypeURLPrefix + "DebugInfo"
	DetailBadRequest   = detailTypeURLPrefix + "BadRequest"
	DetailRequestInfo  = detailTypeURLPrefix + "RequestInfo"
	DetailResourceInfo = detailTypeURLPrefix + "ResourceInfo"
)

// All error detail types implement the Detail interface by exposing TypeURL() string.
// https://github.com/googleapis/googleapis/blob/master/google/rpc/error_details.proto
type Detail interface {
	TypeURL() string
}

// ErrorInfo identifies domain error reason and metadata.
type ErrorInfo struct {
	Type     string            `json:"@type"`
	Reason   string            `json:"reason"`
	Domain   string            `json:"domain"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

func (d *ErrorInfo) TypeURL() string { return d.Type }

var (
	errorInfoReasonMaxLength  = 63
	errorInfoMetaKeyMaxLength = 64
	errorInfoReasonRegex      = regexp.MustCompile(`^[A-Z][A-Z0-9_]+[A-Z0-9]$`)
	errorInfoMetaKeyRegex     = regexp.MustCompile(`^[a-z][a-zA-Z0-9-_]+$`)
)

// Constructs *ErrorInfo.
// Auto-populates package default domain if empty.
// Defaults invalid/empty reason to CodeInternal.Name(), and strips metadata keys violating format constraints.
func NewErrorInfo(reason, domain string, metadata map[string]string) *ErrorInfo {
	if domain == "" {
		domain = getDomain()
	}

	if reason == "" || !isValidErrorInfoReason(reason) {
		reason = CodeInternal.Name()
	}

	var finalMetadata map[string]string

	for k, v := range metadata {
		if isValidErrorInfoMetaKey(k) {
			if finalMetadata == nil {
				finalMetadata = make(map[string]string)
			}
			finalMetadata[k] = v
		}
	}

	return &ErrorInfo{
		Type:     DetailErrorInfo,
		Reason:   reason,
		Domain:   domain,
		Metadata: finalMetadata,
	}
}

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

// RetryInfo describes when client can retry failed request.
type RetryInfo struct {
	Type       string `json:"@type"`
	RetryDelay string `json:"retryDelay"`
}

func (d *RetryInfo) TypeURL() string { return d.Type }

// NewRetryInfo returns *RetryInfo containing duration string representation.
// Returns nil if duration <= 0.
func NewRetryInfo(d time.Duration) *RetryInfo {
	if d <= 0 {
		return nil
	}
	return &RetryInfo{
		Type:       DetailRetryInfo,
		RetryDelay: d.String(),
	}
}

// Duration parses string duration back into time.Duration.
// Returns 0 on invalid string.
func (r *RetryInfo) Duration() time.Duration {
	if r == nil || r.RetryDelay == "" {
		return 0
	}
	d, err := time.ParseDuration(r.RetryDelay)
	if err != nil {
		return 0
	}
	return d
}

// DebugInfo provides debugging trace and stack entry information.
type DebugInfo struct {
	Type         string   `json:"@type"`
	Detail       string   `json:"detail"`
	StackEntries []string `json:"stackEntries,omitempty"`
}

func (d *DebugInfo) TypeURL() string { return d.Type }

// NewDebugInfo constructs *DebugInfo.
// Returns nil if detail is empty.
func NewDebugInfo(detail string, stackEntries ...string) *DebugInfo {
	if detail == "" {
		return nil
	}
	return &DebugInfo{
		Type:         DetailDebugInfo,
		Detail:       detail,
		StackEntries: stackEntries,
	}
}

// FieldViolation describes a client request field validation failure.
type FieldViolation struct {
	Field       string `json:"field"`
	Description string `json:"description"`
	Reason      string `json:"reason,omitempty"`
}

// NewFieldViolation creates field violation detail.
// Returns nil if field or description is empty.
func NewFieldViolation(field, description, reason string) *FieldViolation {
	if field == "" || description == "" {
		return nil
	}
	return &FieldViolation{Field: field, Description: description, Reason: reason}
}

// BadRequest describes client request field validation failures.
type BadRequest struct {
	Type            string           `json:"@type"`
	FieldViolations []FieldViolation `json:"fieldViolations,omitempty"`
}

func (d *BadRequest) TypeURL() string { return d.Type }

// NewBadRequest wraps collection of field violations.
// Returns nil if violations slice is empty.
func NewBadRequest(violations ...FieldViolation) *BadRequest {
	if len(violations) == 0 {
		return nil
	}
	return &BadRequest{Type: DetailBadRequest, FieldViolations: violations}
}

// RequestInfo contains request tracking metadata.
type RequestInfo struct {
	Type        string `json:"@type"`
	RequestId   string `json:"requestId"`
	ServingData string `json:"servingData,omitempty"`
}

func (d *RequestInfo) TypeURL() string { return d.Type }

// NewRequestInfo creates request info object.
// Returns nil if requestID is empty.
func NewRequestInfo(requestID, servingData string) *RequestInfo {
	if requestID == "" {
		return nil
	}
	return &RequestInfo{Type: DetailRequestInfo, RequestId: requestID, ServingData: servingData}
}

// ResourceInfo describes resource affected by operation.
type ResourceInfo struct {
	Type         string `json:"@type"`
	ResourceType string `json:"resourceType"`
	ResourceName string `json:"resourceName"`
	Owner        string `json:"owner,omitempty"`
	Description  string `json:"description,omitempty"`
}

func (d *ResourceInfo) TypeURL() string { return d.Type }

// NewResourceInfo creates resource description object.
// Returns nil if rType or rName is empty.
func NewResourceInfo(rType, rName, owner, desc string) *ResourceInfo {
	if rType == "" || rName == "" {
		return nil
	}
	return &ResourceInfo{Type: DetailResourceInfo, ResourceType: rType, ResourceName: rName, Owner: owner, Description: desc}
}
