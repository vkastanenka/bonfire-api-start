package email

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"log/slog"
	"time"

	"github.com/resend/resend-go/v3"
)

type ResendMailer struct {
	client *resend.Client
	cfg    Config
	tmpl   *template.Template
}

func NewResendMailer(cfg Config) *ResendMailer {
	tmpl, err := LoadTemplates()
	if err != nil {
		panic(fmt.Errorf("failed to parse email templates: %w", err))
	}

	return &ResendMailer{
		client: resend.NewClient(cfg.ResendAPIKey),
		cfg:    cfg,
		tmpl:   tmpl,
	}
}

func (m *ResendMailer) send(ctx context.Context, templateName, recipient, subject string, data map[string]any) error {
	actualRecipient := recipient
	if m.cfg.OverrideTo != "" {
		actualRecipient = m.cfg.OverrideTo
	}

	var body bytes.Buffer
	// Execute the specific template file from the parsed set
	if err := m.tmpl.ExecuteTemplate(&body, templateName, data); err != nil {
		return fmt.Errorf("failed to execute email template: %w", err)
	}

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context cancelled before email dispatch: %w", err)
	}

	// Wrap network call in a timeout
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
		return fmt.Errorf("failed to dispatch email via Resend: %w", err)
	}

	slog.Info("email dispatched via resend", "to", actualRecipient, "subject", subject, "resend_id", resp.Id)
	return nil
}

func (m *ResendMailer) SendRegisterEmail(ctx context.Context, emailAddress, username, token string) error {
	return m.send(ctx, "register.html", emailAddress, "Welcome to Bonfire! 🔥", map[string]any{
		"Title":      fmt.Sprintf("Welcome to Bonfire, %s! 🔥", username),
		"Message":    "We're excited to have you. Before you can start joining servers, you need to verify your email.",
		"ActionText": "Verify Email",
		"ActionLink": fmt.Sprintf("%s/verify?token=%s", m.cfg.FrontendURL, token),
	})
}

func (m *ResendMailer) SendResendVerificationEmail(ctx context.Context, emailAddress, username, token string) error {
    return m.send(ctx, "resend_verification.html", emailAddress, "Verification Request for Bonfire", map[string]any{
        "Title":      "Verification Email Requested",
        "Message":    "We received a request to resend your verification email for your Bonfire account. Use the button below to verify your email address and join the community.",
        "ActionText": "Verify Email",
        "ActionLink": fmt.Sprintf("%s/verify?token=%s", m.cfg.FrontendURL, token),
    })
}

func (m *ResendMailer) SendPasswordResetEmail(ctx context.Context, emailAddress, resetToken string) error {
	return m.send(ctx, "forgot_password.html", emailAddress, "Reset your Bonfire password", map[string]any{
		"Title":      "Password Reset Request",
		"Message":    "We received a request to reset your password. If you didn't request this, ignore this email.",
		"ActionText": "Reset Password",
		"ActionLink": fmt.Sprintf("%s/reset-password?token=%s", m.cfg.FrontendURL, resetToken),
	})
}
