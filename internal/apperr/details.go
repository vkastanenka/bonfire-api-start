package apperr

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// https://github.com/googleapis/googleapis/blob/master/google/rpc/error_details.proto

const (
	DetailErrorInfo           = "type.googleapis.com/google.rpc.ErrorInfo"
	DetailRetryInfo           = "type.googleapis.com/google.rpc.RetryInfo"
	DetailDebugInfo           = "type.googleapis.com/google.rpc.DebugInfo"
	DetailQuotaFailure        = "type.googleapis.com/google.rpc.QuotaFailure"
	DetailPreconditionFailure = "type.googleapis.com/google.rpc.PreconditionFailure"
	DetailBadRequest          = "type.googleapis.com/google.rpc.BadRequest"
	DetailRequestInfo         = "type.googleapis.com/google.rpc.RequestInfo"
	DetailResourceInfo        = "type.googleapis.com/google.rpc.ResourceInfo"
	DetailHelp                = "type.googleapis.com/google.rpc.Help"
	DetailLocalizedMessage    = "type.googleapis.com/google.rpc.LocalizedMessage"
)

type Detail interface {
	TypeURL() string
}

type RawDetail struct {
	Type    string          `json:"@type"`
	RawData json.RawMessage `json:"-"`
}

func (r *RawDetail) TypeURL() string { return r.Type }

// --- ErrorInfo ---

type ErrorInfo struct {
	Type     string            `json:"@type"`
	Reason   string            `json:"reason"`
	Domain   string            `json:"domain"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

func NewErrorInfo(reason, domain string, metadata map[string]string) (*ErrorInfo, error) {
	if reason == "" || domain == "" {
		return nil, errors.New("apperr: ErrorInfo reason and domain are required")
	}
	return &ErrorInfo{Type: DetailErrorInfo, Reason: reason, Domain: domain, Metadata: metadata}, nil
}

func (e *ErrorInfo) TypeURL() string { return DetailErrorInfo }

// --- RetryInfo ---

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

// --- DebugInfo ---

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

// --- QuotaFailure ---

type QuotaViolation struct {
	Subject          string            `json:"subject,omitempty"`
	Description      string            `json:"description,omitempty"`
	ApiService       string            `json:"apiService,omitempty"`
	QuotaMetric      string            `json:"quotaMetric,omitempty"`
	QuotaId          string            `json:"quotaId,omitempty"`
	QuotaDimensions  map[string]string `json:"quotaDimensions,omitempty"`
	QuotaValue       *int64            `json:"quotaValue,omitempty"`
	FutureQuotaValue *int64            `json:"futureQuotaValue,omitempty"`
}

type QuotaFailure struct {
	Type       string           `json:"@type"`
	Violations []QuotaViolation `json:"violations,omitempty"`
}

func NewQuotaFailure(violations ...QuotaViolation) (*QuotaFailure, error) {
	if len(violations) == 0 {
		return nil, errors.New("apperr: QuotaFailure requires at least one violation")
	}
	return &QuotaFailure{Type: DetailQuotaFailure, Violations: violations}, nil
}

func (q *QuotaFailure) TypeURL() string { return DetailQuotaFailure }

// --- PreconditionFailure ---

type PreconditionViolation struct {
	Type        string `json:"type,omitempty"`
	Subject     string `json:"subject,omitempty"`
	Description string `json:"description,omitempty"`
}

func NewPreconditionViolation(vType, subject, description string) *PreconditionViolation {
	return &PreconditionViolation{Type: vType, Subject: subject, Description: description}
}

type PreconditionFailure struct {
	Type       string                  `json:"@type"`
	Violations []PreconditionViolation `json:"violations,omitempty"`
}

func NewPreconditionFailure(violations ...PreconditionViolation) (*PreconditionFailure, error) {
	if len(violations) == 0 {
		return nil, errors.New("apperr: PreconditionFailure requires at least one violation")
	}
	return &PreconditionFailure{Type: DetailPreconditionFailure, Violations: violations}, nil
}

func (p *PreconditionFailure) TypeURL() string { return DetailPreconditionFailure }

// --- BadRequest ---

type FieldViolation struct {
	Field            string            `json:"field"`
	Description      string            `json:"description"`
	Reason           string            `json:"reason"`
	LocalizedMessage *LocalizedMessage `json:"localizedMessage,omitempty"`
}

func NewFieldViolation(field, description, reason string, lm *LocalizedMessage) (*FieldViolation, error) {
	if field == "" || description == "" || reason == "" {
		return nil, errors.New("apperr: FieldViolation field, description, and reason are required")
	}
	return &FieldViolation{Field: field, Description: description, Reason: reason, LocalizedMessage: lm}, nil
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

// --- RequestInfo ---

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

// --- ResourceInfo ---

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

// --- Help ---

type HelpLink struct {
	Description string `json:"description"`
	URL         string `json:"url"`
}

func NewHelpLink(description, rawURL string) (*HelpLink, error) {
	if description == "" || rawURL == "" {
		return nil, errors.New("apperr: HelpLink description and URL are required")
	}
	return &HelpLink{Description: description, URL: rawURL}, nil
}

type Help struct {
	Type  string     `json:"@type"`
	Links []HelpLink `json:"links,omitempty"`
}

func NewHelp(links ...HelpLink) (*Help, error) {
	if len(links) == 0 {
		return nil, errors.New("apperr: Help requires at least one link")
	}
	return &Help{Type: DetailHelp, Links: links}, nil
}

func (h *Help) TypeURL() string { return DetailHelp }

// --- LocalizedMessage ---

type LocalizedMessage struct {
	Type    string `json:"@type"`
	Locale  string `json:"locale"`
	Message string `json:"message"`
}

func NewLocalizedMessage(locale, message string) (*LocalizedMessage, error) {
	if locale == "" || message == "" {
		return nil, errors.New("apperr: LocalizedMessage locale and message are required")
	}
	return &LocalizedMessage{Type: DetailLocalizedMessage, Locale: locale, Message: message}, nil
}

func (l *LocalizedMessage) TypeURL() string { return DetailLocalizedMessage }
