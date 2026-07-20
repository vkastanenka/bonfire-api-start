package apperr

import (
	"log/slog"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type Option func(*Error)

var (
	metaKeyRegex = regexp.MustCompile(`^[a-z][a-zA-Z0-9-_]*$`)
	reasonRegex  = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)
)

var (
	htmlRegex    = regexp.MustCompile(`<[^>]*>`)
	mdLinkRegex  = regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`)
	mdStyleRegex = regexp.MustCompile(`[\*_~` + "`" + `]`)
)

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func WithError(err error) Option {
	return func(e *Error) { e.Err = err }
}

func WithMeta(key, value string) Option {
	return func(e *Error) {
		if len(key) > 64 {
			slog.Warn("apperr: metadata key exceeds 64 characters", "key", key)
			return
		}

		if !metaKeyRegex.MatchString(key) {
			slog.Error("apperr: metadata key dropped; validation failed: Must start with a lowercase letter and contain only lowercase letters, numbers, hyphens, or underscores.", "key", key, "value", value)
			return
		}

		if e.ErrorInfo == nil {
			e.ErrorInfo = &ErrorInfo{}
		}
		if e.ErrorInfo.Metadata == nil {
			e.ErrorInfo.Metadata = make(map[string]string)
		}
		e.ErrorInfo.Metadata[key] = value
	}
}

func WithParams(params map[string]string) Option {
	return func(e *Error) {
		if len(params) == 0 {
			return
		}

		for k, v := range params {
			placeholder := "{" + k + "}"

			e.Message = strings.ReplaceAll(e.Message, placeholder, v)

			if e.LocalizedMessage != nil {
				e.LocalizedMessage.Message = strings.ReplaceAll(e.LocalizedMessage.Message, placeholder, v)
			}

			WithMeta(k, v)(e)
		}
	}
}

func WithRetryInfo(delay time.Duration) Option {
	return func(e *Error) {
		e.RetryInfo = &RetryInfo{RetryDelay: delay}
	}
}

func WithDebugInfo(detail string, stack []string) Option {
	return func(e *Error) {
		e.DebugInfo = &DebugInfo{
			Detail:       detail,
			StackEntries: stack,
		}
	}
}

func WithQuotaViolation(violation QuotaViolation) Option {
	return func(e *Error) {
		if e.QuotaFailure == nil {
			e.QuotaFailure = &QuotaFailure{}
		}
		e.QuotaFailure.Violations = append(e.QuotaFailure.Violations, violation)
	}
}

func WithPreconditionViolation(vType, subject, description string) Option {
	return func(e *Error) {
		if e.PreconditionFailure == nil {
			e.PreconditionFailure = &PreconditionFailure{}
		}
		e.PreconditionFailure.Violations = append(e.PreconditionFailure.Violations, PreconditionViolation{
			Type:        vType,
			Subject:     subject,
			Description: description,
		})
	}
}

func WithFieldViolation(field, description, reason string) Option {
	return func(e *Error) {
		if e.BadRequest == nil {
			e.BadRequest = &BadRequest{}
		}

		if reason != "" && !reasonRegex.MatchString(reason) {
			slog.Error("apperr: FieldViolation reason dropped; validation failed: Must be UPPER_SNAKE_CASE.", "field", field, "invalid_reason", reason)
			return
		}

		e.BadRequest.FieldViolations = append(e.BadRequest.FieldViolations, FieldViolation{
			Field:       field,
			Description: description,
			Reason:      strPtr(reason),
		})
	}
}

func WithRequestInfo(id, servingData string) Option {
	return func(e *Error) {
		e.RequestInfo = &RequestInfo{
			RequestId:   id,
			ServingData: strPtr(servingData),
		}
	}
}

func WithResourceInfo(rType, name, owner, description string) Option {
	return func(e *Error) {
		e.ResourceInfo = &ResourceInfo{
			ResourceType: rType,
			ResourceName: name,
			Owner:        strPtr(owner),
			Description:  strPtr(description),
		}
	}
}

func WithHelpLink(description, rawURL string) Option {
	return func(e *Error) {
		if rawURL == "" || description == "" {
			return
		}

		hasHTML := htmlRegex.MatchString(description)
		hasMDLink := mdLinkRegex.MatchString(description)
		hasMDStyle := mdStyleRegex.MatchString(description)

		if hasHTML || hasMDLink || hasMDStyle {
			slog.Error("apperr: HelpLink description dropped; validation failed: Must be plain text.", "invalid_description", description)
			return
		}

		parsedURL, err := url.Parse(rawURL)
		if err != nil || !parsedURL.IsAbs() {
			slog.Error("apperr: HelpLink URL dropped; validation failed: Must be an absolute URL.", "invalid_url", rawURL)
			return
		}

		if e.Help == nil {
			e.Help = &Help{}
		}

		e.Help.Links = append(e.Help.Links, HelpLink{
			Description: description,
			URL:         parsedURL.String(),
		})
	}
}

func WithLocalizedMessage(locale, message string) Option {
	return func(e *Error) {
		lm := &LocalizedMessage{
			Locale:  locale,
			Message: message,
		}

		if e.ErrorInfo != nil && len(e.ErrorInfo.Metadata) > 0 {
			for k, v := range e.ErrorInfo.Metadata {
				placeholder := "{" + k + "}"
				lm.Message = strings.ReplaceAll(lm.Message, placeholder, v)
			}
		}

		e.LocalizedMessage = lm
	}
}
