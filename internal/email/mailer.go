package email

import (
	"bonfire-api/internal/pkg/errs"
	"bytes"
	"context"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"time"

	"github.com/resend/resend-go/v3"
)

//go:embed templates/*.html
var emailTemplates embed.FS

func LoadTemplates() (*template.Template, error) {
	return template.ParseFS(emailTemplates, "templates/*.html")
}

type Config struct {
	ResendAPIKey string
	FromAddress  string
	FrontendURL  string
	OverrideTo   string
}

type Mailer struct {
	client *resend.Client
	cfg    Config
	tmpl   *template.Template
}

func NewMailer(cfg Config) *Mailer {
	tmpl, err := LoadTemplates()
	if err != nil {
		panic(fmt.Errorf("failed to parse email templates: %w", err))
	}

	if cfg.OverrideTo != "" {
		slog.Info("email engine: Resend initialized in SANDBOX mode", "override_to", cfg.OverrideTo)
	} else {
		slog.Info("email engine: Resend initialized in PRODUCTION mode")
	}

	return &Mailer{
		client: resend.NewClient(cfg.ResendAPIKey),
		cfg:    cfg,
		tmpl:   tmpl,
	}
}

func (m *Mailer) send(ctx context.Context, templateName, recipient, subject string, data any) error {
	actualRecipient := recipient
	if m.cfg.OverrideTo != "" {
		actualRecipient = m.cfg.OverrideTo
	}

	var body bytes.Buffer
	if err := m.tmpl.ExecuteTemplate(&body, templateName, data); err != nil {
		return errs.Internal("Failed to execute email template.").Wrap(err)
	}

	if err := ctx.Err(); err != nil {
		if errors.Is(err, context.Canceled) {
			return errs.Cancelled("Context was cancelled before email dispatch.").Wrap(err)
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return errs.DeadlineExceeded("Context deadline was exceeded before email dispatch.").Wrap(err)
		}
		return errs.Internal("Context was closed before email dispatch.").Wrap(err)
	}

	_, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	params := &resend.SendEmailRequest{
		From:    m.cfg.FromAddress,
		To:      []string{actualRecipient},
		Subject: subject,
		Html:    body.String(),
	}

	resp, err := m.client.Emails.Send(params)
	if err != nil {
		return errs.Internal("Failed to dispatch email via Resend.").Wrap(err)
	}

	slog.Info("email dispatched via resend",
		slog.String("to", actualRecipient),
		slog.String("subject", subject),
		slog.String("resend_id", resp.Id),
	)
	return nil
}
