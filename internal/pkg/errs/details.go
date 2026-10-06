package errs

import (
	"regexp"
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

type Detail interface {
	TypeURL() string
}

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

type RetryInfo struct {
	Type       string `json:"@type"`
	RetryDelay string `json:"retryDelay"`
}

func (d *RetryInfo) TypeURL() string { return d.Type }

func NewRetryInfo(d time.Duration) *RetryInfo {
	if d <= 0 {
		return nil
	}
	return &RetryInfo{
		Type:       DetailRetryInfo,
		RetryDelay: d.String(),
	}
}

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

type DebugInfo struct {
	Type         string   `json:"@type"`
	Detail       string   `json:"detail"`
	StackEntries []string `json:"stackEntries,omitempty"`
}

func (d *DebugInfo) TypeURL() string { return d.Type }

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

type FieldViolation struct {
	Field       string `json:"field"`
	Description string `json:"description"`
	Reason      string `json:"reason,omitempty"`
}

func NewFieldViolation(field, description, reason string) *FieldViolation {
	if field == "" || description == "" {
		return nil
	}
	return &FieldViolation{Field: field, Description: description, Reason: reason}
}

type BadRequest struct {
	Type            string           `json:"@type"`
	FieldViolations []FieldViolation `json:"fieldViolations"`
}

func (d *BadRequest) TypeURL() string { return d.Type }

func NewBadRequest(violations ...FieldViolation) *BadRequest {
	if len(violations) == 0 {
		return nil
	}
	return &BadRequest{Type: DetailBadRequest, FieldViolations: violations}
}

type RequestInfo struct {
	Type        string `json:"@type"`
	RequestId   string `json:"requestId"`
	ServingData string `json:"servingData,omitempty"`
}

func (d *RequestInfo) TypeURL() string { return d.Type }

func NewRequestInfo(requestID, servingData string) *RequestInfo {
	if requestID == "" {
		return nil
	}
	return &RequestInfo{Type: DetailRequestInfo, RequestId: requestID, ServingData: servingData}
}

type ResourceInfo struct {
	Type         string `json:"@type"`
	ResourceType string `json:"resourceType"`
	ResourceName string `json:"resourceName"`
	Owner        string `json:"owner,omitempty"`
	Description  string `json:"description,omitempty"`
}

func (d *ResourceInfo) TypeURL() string { return d.Type }

func NewResourceInfo(rType, rName, owner, desc string) *ResourceInfo {
	if rType == "" || rName == "" {
		return nil
	}
	return &ResourceInfo{Type: DetailResourceInfo, ResourceType: rType, ResourceName: rName, Owner: owner, Description: desc}
}

