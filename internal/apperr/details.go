package apperr

import (
	"time"
)

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

type ErrorInfo struct {
	Reason   string            `json:"reason"`
	Domain   string            `json:"domain"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type RetryInfo struct {
	RetryDelay time.Duration `json:"-"`
}

type DebugInfo struct {
	StackEntries []string `json:"stackEntries,omitempty"`
	Detail       string   `json:"detail,omitempty"`
}

type QuotaViolation struct {
	Subject          string            `json:"subject"`
	Description      string            `json:"description"`
	ApiService       string            `json:"apiService"`
	QuotaMetric      string            `json:"quotaMetric"`
	QuotaId          string            `json:"quotaId"`
	QuotaDimensions  map[string]string `json:"quotaDimensions,omitempty"`
	QuotaValue       *int64            `json:"quotaValue,omitempty"`
	FutureQuotaValue *int64            `json:"futureQuotaValue,omitempty"`
}

type QuotaFailure struct {
	Violations []QuotaViolation `json:"violations,omitempty"`
}

type PreconditionViolation struct {
	Type        string `json:"type"`
	Subject     string `json:"subject"`
	Description string `json:"description"`
}

type PreconditionFailure struct {
	Violations []PreconditionViolation `json:"violations,omitempty"`
}

type FieldViolation struct {
	Field            string            `json:"field"`
	Description      string            `json:"description"`
	Reason           *string           `json:"reason,omitempty"`
	LocalizedMessage *LocalizedMessage `json:"localizedMessage,omitempty"`
}

type BadRequest struct {
	FieldViolations []FieldViolation `json:"fieldViolations,omitempty"`
}

type RequestInfo struct {
	RequestId   string  `json:"requestId"`
	ServingData *string `json:"servingData,omitempty"`
}

type ResourceInfo struct {
	ResourceType string  `json:"resourceType"`
	ResourceName string  `json:"resourceName"`
	Owner        *string `json:"owner,omitempty"`
	Description  *string `json:"description,omitempty"`
}

type HelpLink struct {
	Description string `json:"description"`
	URL         string `json:"url"`
}

type Help struct {
	Links []HelpLink `json:"links,omitempty"`
}

type LocalizedMessage struct {
	Locale  string `json:"locale"`
	Message string `json:"message"`
}
