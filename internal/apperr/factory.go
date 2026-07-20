package apperr

import (
	"log/slog"
	"net/url"
	"strings"
)

func New(code Code, reason string, message string, opts ...Option) error {
	if message == "" {
		message = code.Message()
	}

	finalReason := reason
	if finalReason == "" {
		finalReason = code.String()
	} else {
		if len(finalReason) > 63 {
			slog.Error("apperr: ErrorInfo.reason validation failed: Must be at most 63 characters.", "reason", finalReason)
			opts = append(opts, WithMeta("x-apperr-compliance-violation", "true"))
		}

		if !reasonRegex.MatchString(finalReason) {
			slog.Error("apperr: ErrorInfo.reason validation failed: Must be UPPER_SNAKE_CASE (start with uppercase letter, contain only uppercase letters, numbers, or underscores).", "reason", finalReason)
			opts = append(opts, WithMeta("x-apperr-compliance-violation", "true"))
		}
	}

	e := &Error{
		Code:    code,
		Message: message,
	}

	e.ErrorInfo = &ErrorInfo{
		Domain: getDefaultDomain(),
		Reason: finalReason,
	}

	for _, opt := range opts {
		opt(e)
	}

	if e.Help != nil && e.ErrorInfo != nil && e.ErrorInfo.Reason != "" {
		for i, link := range e.Help.Links {
			if u, err := url.Parse(link.URL); err == nil && u.Fragment == "" {
				u.Fragment = e.ErrorInfo.Reason
				e.Help.Links[i].URL = u.String()
			}
		}
	}

	if len(e.Message) > 0 && strings.Contains(e.Message, "%") {
		slog.Error("apperr: Raw format verb detected. Use WithParams instead of inline formatting.", "msg", e.Message)
		WithMeta("x-apperr-compliance-violation", "true")(e)
	}

	return e
}

func NewCancelled(err error, reason string, msg string, opts ...Option) error {
	return New(CodeCancelled, reason, msg, append(opts, WithError(err))...)
}
func NewUnknown(err error, reason string, msg string, opts ...Option) error {
	return New(CodeUnknown, reason, msg, append(opts, WithError(err))...)
}
func NewInvalidArgument(err error, reason string, msg string, opts ...Option) error {
	return New(CodeInvalidArgument, reason, msg, append(opts, WithError(err))...)
}
func NewDeadlineExceeded(err error, reason string, msg string, opts ...Option) error {
	return New(CodeDeadlineExceeded, reason, msg, append(opts, WithError(err))...)
}
func NewNotFound(err error, reason string, msg string, opts ...Option) error {
	return New(CodeNotFound, reason, msg, append(opts, WithError(err))...)
}
func NewAlreadyExists(err error, reason string, msg string, opts ...Option) error {
	return New(CodeAlreadyExists, reason, msg, append(opts, WithError(err))...)
}
func NewPermissionDenied(err error, reason string, msg string, opts ...Option) error {
	return New(CodePermissionDenied, reason, msg, append(opts, WithError(err))...)
}
func NewResourceExhausted(err error, reason string, msg string, opts ...Option) error {
	return New(CodeResourceExhausted, reason, msg, append(opts, WithError(err))...)
}
func NewFailedPrecondition(err error, reason string, msg string, opts ...Option) error {
	return New(CodeFailedPrecondition, reason, msg, append(opts, WithError(err))...)
}
func NewAborted(err error, reason string, msg string, opts ...Option) error {
	return New(CodeAborted, reason, msg, append(opts, WithError(err))...)
}
func NewOutOfRange(err error, reason string, msg string, opts ...Option) error {
	return New(CodeOutOfRange, reason, msg, append(opts, WithError(err))...)
}
func NewUnimplemented(err error, reason string, msg string, opts ...Option) error {
	return New(CodeUnimplemented, reason, msg, append(opts, WithError(err))...)
}
func NewInternal(err error, reason string, msg string, opts ...Option) error {
	return New(CodeInternal, reason, msg, append(opts, WithError(err))...)
}
func NewUnavailable(err error, reason string, msg string, opts ...Option) error {
	return New(CodeUnavailable, reason, msg, append(opts, WithError(err))...)
}
func NewDataLoss(err error, reason string, msg string, opts ...Option) error {
	return New(CodeDataLoss, reason, msg, append(opts, WithError(err))...)
}
func NewUnauthenticated(err error, reason string, msg string, opts ...Option) error {
	return New(CodeUnauthenticated, reason, msg, append(opts, WithError(err))...)
}
