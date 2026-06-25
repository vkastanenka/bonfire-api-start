package email

import (
	"context"
	"log/slog"
)

// --- MAILER TYPES ---

// Mailer
type Mailer interface {
	SendRegisterEmail(ctx context.Context, emailAddress, username, token string) error
	SendResendVerificationEmail(ctx context.Context, emailAddress, username, token string) error
	SendPasswordResetEmail(ctx context.Context, emailAddress, resetToken string) error
}

// NoOpMailer
type NoOpMailer struct{}

func (n *NoOpMailer) SendRegisterEmail(ctx context.Context, e, u, t string) error { return nil }
func (n *NoOpMailer) SendResendVerificationEmail(ctx context.Context, e, u, t string) error {
	return nil
}
func (n *NoOpMailer) SendPasswordResetEmail(ctx context.Context, e, t string) error { return nil }

// Config
type Config struct {
	ResendAPIKey string
	FromAddress  string
	FrontendURL  string
	OverrideTo   string
}

// --- MAILER INITIALIZATION ---

// NewMailer
func NewMailer(cfg Config) Mailer {
	if cfg.ResendAPIKey == "" {
		slog.Warn("email engine: API Key missing, defaulting to No-Op mailer")
		return &NoOpMailer{}
	}

	if cfg.OverrideTo != "" {
		slog.Info("email engine: Resend initialized in SANDBOX mode", "override_to", cfg.OverrideTo)
	} else {
		slog.Info("email engine: Resend initialized in PRODUCTION mode")
	}
	return NewResendMailer(cfg)
}
