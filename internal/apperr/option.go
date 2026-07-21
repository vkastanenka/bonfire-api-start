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

func WithOptions(options ...Option) Option {
	return func(e *Error) {
		for _, opt := range options {
			if opt != nil {
				opt(e)
			}
		}
	}
}

func WithDetail(d Detail) Option {
	return WithDetails(d)
}

func WithDetails(details ...Detail) Option {
	return func(e *Error) {
		for _, d := range details {
			e.addDetail(d)
		}
	}
}

func WithError(err error) Option {
	return func(e *Error) {
		e.Err = err
	}
}

func WithMsg(message string) Option {
	return func(e *Error) {
		e.Message = message
	}
}

func WithReason(reason string) Option {
	return func(e *Error) {
		if info, ok := e.GetDetail(DetailErrorInfo).(*ErrorInfo); ok && info != nil {
			info.Reason = reason
		}
	}
}

func WithMeta(key, value string) Option {
	return func(e *Error) {
		if len(key) > 64 {
			slog.Warn("apperr: metadata key exceeds maximum length of 64 characters", "key", key)
			return
		}
		if info, ok := e.GetDetail(DetailErrorInfo).(*ErrorInfo); ok && info != nil {
			if info.Metadata == nil {
				info.Metadata = make(map[string]string)
			}
			info.Metadata[key] = value
		}
	}
}

func WithRetryInfo(delay time.Duration) Option {
	return WithDetail(NewRetryInfo(delay))
}

func WithDebugInfo(detail string, stack ...string) Option {
	info, err := NewDebugInfo(detail, stack...)
	if err != nil {
		slog.Warn("apperr: failed to create DebugInfo", "error", err)
		return func(*Error) {}
	}
	return WithDetail(info)
}

func WithQuotaViolation(subject, description string) Option {
	qf, err := NewQuotaFailure(QuotaViolation{
		Subject:     subject,
		Description: description,
	})
	if err != nil {
		slog.Warn("apperr: failed to create QuotaFailure", "error", err)
		return func(*Error) {}
	}
	return WithDetail(qf)
}

func WithPreconditionViolation(vType, subject, description string) Option {
	pv := NewPreconditionViolation(vType, subject, description)
	pf, err := NewPreconditionFailure(*pv)
	if err != nil {
		slog.Warn("apperr: failed to create PreconditionFailure", "error", err)
		return func(*Error) {}
	}
	return WithDetail(pf)
}

func WithFieldViolation(field, description, reason string) Option {
	fv, err := NewFieldViolation(field, description, reason, nil)
	if err != nil {
		slog.Warn("apperr: failed to create FieldViolation", "error", err)
		return func(*Error) {}
	}

	br, err := NewBadRequest(*fv)
	if err != nil {
		slog.Warn("apperr: failed to create BadRequest", "error", err)
		return func(*Error) {}
	}
	return WithDetail(br)
}

func WithRequestInfo(id, servingData string) Option {
	info, err := NewRequestInfo(id, servingData)
	if err != nil {
		slog.Warn("apperr: failed to create RequestInfo", "error", err)
		return func(*Error) {}
	}
	return WithDetail(info)
}

func WithResourceInfo(rType, name, owner, description string) Option {
	info, err := NewResourceInfo(rType, name, owner, description)
	if err != nil {
		slog.Warn("apperr: failed to create ResourceInfo", "error", err)
		return func(*Error) {}
	}
	return WithDetail(info)
}

func WithHelpLink(description, rawURL string) Option {
	if rawURL == "" || description == "" {
		return func(*Error) {}
	}

	if !isPlainText(description) {
		slog.Error("apperr: HelpLink description must be plain text", "description", description)
		return func(*Error) {}
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil || !parsedURL.IsAbs() {
		slog.Error("apperr: HelpLink URL must be absolute", "url", rawURL)
		return func(*Error) {}
	}

	link, err := NewHelpLink(description, parsedURL.String())
	if err != nil {
		slog.Warn("apperr: failed to create HelpLink", "error", err)
		return func(*Error) {}
	}

	h, err := NewHelp(*link)
	if err != nil {
		slog.Warn("apperr: failed to create Help", "error", err)
		return func(*Error) {}
	}

	return WithDetail(h)
}

func WithLocalizedMessage(locale, message string) Option {
	lm, err := NewLocalizedMessage(locale, message)
	if err != nil {
		slog.Warn("apperr: failed to create LocalizedMessage", "error", err)
		return func(*Error) {}
	}
	return WithDetail(lm)
}
