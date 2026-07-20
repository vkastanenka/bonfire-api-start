package apperr

import (
	"encoding/json"
	"fmt"
)

var _ json.Marshaler = (*Error)(nil)

func (e *Error) MarshalJSON() ([]byte, error) {
	details := make([]interface{}, 0)

	if e.ErrorInfo != nil {
		domain := e.ErrorInfo.Domain
		if domain == "" {
			domain = getDefaultDomain()
		}

		details = append(details, struct {
			Type   string            `json:"@type"`
			Domain string            `json:"domain"`
			Reason string            `json:"reason"`
			Meta   map[string]string `json:"metadata,omitempty"`
		}{
			Type:   DetailErrorInfo,
			Domain: domain,
			Reason: e.ErrorInfo.Reason,
			Meta:   e.ErrorInfo.Metadata,
		})
	}

	if e.RetryInfo != nil && e.RetryInfo.RetryDelay > 0 {
		details = append(details, struct {
			Type       string `json:"@type"`
			RetryDelay string `json:"retryDelay"`
		}{
			Type:       DetailRetryInfo,
			RetryDelay: fmt.Sprintf("%.3fs", e.RetryInfo.RetryDelay.Seconds()),
		})
	}

	if e.DebugInfo != nil && (len(e.DebugInfo.StackEntries) > 0 || e.DebugInfo.Detail != "") {
		details = append(details, struct {
			Type         string   `json:"@type"`
			Detail       string   `json:"detail,omitempty"`
			StackEntries []string `json:"stackEntries,omitempty"`
		}{
			Type:         DetailDebugInfo,
			Detail:       e.DebugInfo.Detail,
			StackEntries: e.DebugInfo.StackEntries,
		})
	}

	if e.QuotaFailure != nil && len(e.QuotaFailure.Violations) > 0 {
		details = append(details, struct {
			Type       string           `json:"@type"`
			Violations []QuotaViolation `json:"violations"`
		}{
			Type:       DetailQuotaFailure,
			Violations: e.QuotaFailure.Violations,
		})
	}

	if e.PreconditionFailure != nil && len(e.PreconditionFailure.Violations) > 0 {
		details = append(details, struct {
			Type       string                  `json:"@type"`
			Violations []PreconditionViolation `json:"violations"`
		}{
			Type:       DetailPreconditionFailure,
			Violations: e.PreconditionFailure.Violations,
		})
	}

	if e.BadRequest != nil && len(e.BadRequest.FieldViolations) > 0 {
		details = append(details, struct {
			Type            string           `json:"@type"`
			FieldViolations []FieldViolation `json:"fieldViolations"`
		}{
			Type:            DetailBadRequest,
			FieldViolations: e.BadRequest.FieldViolations,
		})
	}

	if e.RequestInfo != nil && e.RequestInfo.RequestId != "" {
		details = append(details, struct {
			Type        string  `json:"@type"`
			RequestId   string  `json:"requestId"`
			ServingData *string `json:"servingData,omitempty"`
		}{
			Type:        DetailRequestInfo,
			RequestId:   e.RequestInfo.RequestId,
			ServingData: e.RequestInfo.ServingData,
		})
	}

	if e.ResourceInfo != nil && e.ResourceInfo.ResourceName != "" {
		details = append(details, struct {
			Type         string  `json:"@type"`
			ResourceType string  `json:"resourceType"`
			ResourceName string  `json:"resourceName"`
			Owner        *string `json:"owner,omitempty"`
			Description  *string `json:"description,omitempty"`
		}{
			Type:         DetailResourceInfo,
			ResourceType: e.ResourceInfo.ResourceType,
			ResourceName: e.ResourceInfo.ResourceName,
			Owner:        e.ResourceInfo.Owner,
			Description:  e.ResourceInfo.Description,
		})
	}

	if e.Help != nil && len(e.Help.Links) > 0 {
		details = append(details, struct {
			Type  string     `json:"@type"`
			Links []HelpLink `json:"links"`
		}{
			Type:  DetailHelp,
			Links: e.Help.Links,
		})
	}

	if e.LocalizedMessage != nil && e.LocalizedMessage.Locale != "" && e.LocalizedMessage.Message != "" {
		details = append(details, struct {
			Type    string `json:"@type"`
			Locale  string `json:"locale"`
			Message string `json:"message"`
		}{
			Type:    DetailLocalizedMessage,
			Locale:  e.LocalizedMessage.Locale,
			Message: e.LocalizedMessage.Message,
		})
	}

	type payload struct {
		Code    int           `json:"code"`
		Message string        `json:"message"`
		Details []interface{} `json:"details"`
	}

	return json.Marshal(payload{
		Code:    int(e.Code),
		Message: e.Message,
		Details: details,
	})
}

func (r *RetryInfo) MarshalJSON() ([]byte, error) {
	if r == nil {
		return []byte("null"), nil
	}
	return json.Marshal(struct {
		RetryDelay string `json:"retryDelay"`
	}{
		RetryDelay: fmt.Sprintf("%.3fs", r.RetryDelay.Seconds()),
	})
}
