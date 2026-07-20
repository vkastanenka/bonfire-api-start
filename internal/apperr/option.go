package apperr

import (
	"log/slog"
	"net/url"
	"strings"
	"time"
)

type Option func(*Error)

func isPlainText(s string) bool {
	return !strings.ContainsAny(s, "<>[]*_~`")
}

func WithDetail(d Detail) Option {
	return func(e *Error) {
		e.addDetail(d)
	}
}

func WithDetails(details ...Detail) Option {
	return func(e *Error) {
		for _, d := range details {
			e.addDetail(d)
		}
	}
}

func WithOptions(options ...Option) Option {
	return func(e *Error) {
		for _, opt := range options {
			if opt != nil {
				opt(e)
			}
		}
	}
}

func WithError(err error) Option {
	return func(e *Error) {
		e.Err = err
	}
}

func WithMessage(message string) Option {
	return func(e *Error) {
		e.Message = message
	}
}

func WithReason(reason string) Option {
	return func(e *Error) {
		if info, ok := e.detMap[DetailErrorInfo].(*ErrorInfo); ok && info != nil {
			info.Reason = reason
			return
		}

		info, err := NewErrorInfo(reason, getDomain(), nil)
		if err != nil {
			slog.Warn("apperr: failed to create ErrorInfo in WithReason", "error", err)
			return
		}
		e.addDetail(info)
	}
}

func WithMeta(key, value string) Option {
	return func(e *Error) {
		if len(key) > 64 {
			slog.Warn("apperr: metadata key exceeds 64 characters", "key", key)
			return
		}

		if info, ok := e.detMap[DetailErrorInfo].(*ErrorInfo); ok && info != nil {
			if info.Metadata == nil {
				info.Metadata = make(map[string]string)
			}
			info.Metadata[key] = value
			return
		}

		// Fallback safeguard in case ErrorInfo wasn't initialized prior to option execution
		info, err := NewErrorInfo("REASON_UNSPECIFIED", getDomain(), map[string]string{key: value})
		if err != nil {
			slog.Warn("apperr: failed to create ErrorInfo in WithMeta", "error", err)
			return
		}
		e.addDetail(info)
	}
}

func WithRetryInfo(delay time.Duration) Option {
	return WithDetail(NewRetryInfo(delay))
}

func WithDebugInfo(detail string, stack ...string) Option {
	info, err := NewDebugInfo(detail, stack...)
	if err != nil {
		slog.Warn("apperr: invalid DebugInfo parameters", "error", err)
		return func(*Error) {}
	}
	return WithDetail(info)
}

func WithQuotaViolation(subject, description string) Option {
	// AIP-193 QuotaViolation is created directly as a struct
	qf, err := NewQuotaFailure(QuotaViolation{
		Subject:     subject,
		Description: description,
	})
	if err != nil {
		slog.Warn("apperr: invalid QuotaFailure parameters", "error", err)
		return func(*Error) {}
	}
	return WithDetail(qf)
}

func WithPreconditionViolation(vType, subject, description string) Option {
	pv := NewPreconditionViolation(vType, subject, description)

	pf, err := NewPreconditionFailure(*pv)
	if err != nil {
		slog.Warn("apperr: invalid PreconditionFailure parameters", "error", err)
		return func(*Error) {}
	}
	return WithDetail(pf)
}

func WithFieldViolation(field, description, reason string) Option {
	fv, err := NewFieldViolation(field, description, reason, nil)
	if err != nil {
		slog.Warn("apperr: invalid FieldViolation parameters", "error", err)
		return func(*Error) {}
	}

	br, err := NewBadRequest(*fv)
	if err != nil {
		slog.Warn("apperr: invalid BadRequest parameters", "error", err)
		return func(*Error) {}
	}
	return WithDetail(br)
}

func WithRequestInfo(id, servingData string) Option {
	info, err := NewRequestInfo(id, servingData)
	if err != nil {
		slog.Warn("apperr: invalid RequestInfo parameters", "error", err)
		return func(*Error) {}
	}
	return WithDetail(info)
}

func WithResourceInfo(rType, name, owner, description string) Option {
	info, err := NewResourceInfo(rType, name, owner, description)
	if err != nil {
		slog.Warn("apperr: invalid ResourceInfo parameters", "error", err)
		return func(*Error) {}
	}
	return WithDetail(info)
}

func WithHelpLink(description, rawURL string) Option {
	if rawURL == "" || description == "" {
		return func(*Error) {}
	}

	if !isPlainText(description) {
		slog.Error("apperr: HelpLink description dropped; must be plain text.", "invalid_description", description)
		return func(*Error) {}
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil || !parsedURL.IsAbs() {
		slog.Error("apperr: HelpLink URL dropped; must be an absolute URL.", "invalid_url", rawURL)
		return func(*Error) {}
	}

	link, err := NewHelpLink(description, parsedURL.String())
	if err != nil {
		slog.Warn("apperr: invalid HelpLink parameters", "error", err)
		return func(*Error) {}
	}

	h, err := NewHelp(*link)
	if err != nil {
		slog.Warn("apperr: invalid Help parameters", "error", err)
		return func(*Error) {}
	}

	return WithDetail(h)
}

func WithLocalizedMessage(locale, message string) Option {
	lm, err := NewLocalizedMessage(locale, message)
	if err != nil {
		slog.Warn("apperr: invalid LocalizedMessage parameters", "error", err)
		return func(*Error) {}
	}
	return WithDetail(lm)
}
