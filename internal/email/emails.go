package email

import (
	"context"
	"fmt"
	"time"
)

type ActionEmailData struct {
	Title       string
	Message     string
	ActionText  string
	ActionLink  string
	RequestedAt string
}

func (c Config) VerificationURL(token string) string {
	return fmt.Sprintf("%s/verify?token=%s", c.FrontendURL, token)
}

func (c Config) PasswordResetURL(token string) string {
	return fmt.Sprintf("%s/reset-password?token=%s", c.FrontendURL, token)
}

func (m *Mailer) SendRegister(ctx context.Context, emailAddress, username, token string, requestedAt time.Time) error {
	return m.sendActionEmail(ctx, "register.html", emailAddress, "Welcome to Bonfire! 🔥", ActionEmailData{
		Title:       fmt.Sprintf("Welcome to Bonfire, %s! 🔥", username),
		Message:     "We're excited to have you. Before you can start joining servers, you need to verify your email.",
		ActionText:  "Verify Email",
		ActionLink:  m.cfg.VerificationURL(token),
		RequestedAt: requestedAt.String(),
	})
}

func (m *Mailer) SendResendVerification(ctx context.Context, emailAddress, username, token string, requestedAt time.Time) error {
	return m.sendActionEmail(ctx, "resend_verification.html", emailAddress, "Verification Request for Bonfire", ActionEmailData{
		Title:       "Verification Email Requested",
		Message:     "We received a request to resend your verification email for your Bonfire account. Use the button below to verify your email address and join the community.",
		ActionText:  "Verify Email",
		ActionLink:  m.cfg.VerificationURL(token),
		RequestedAt: requestedAt.String(),
	})
}

func (m *Mailer) SendPasswordReset(ctx context.Context, emailAddress, resetToken string, requestedAt time.Time) error {
	return m.sendActionEmail(ctx, "forgot_password.html", emailAddress, "Reset your Bonfire password", ActionEmailData{
		Title:       "Password Reset Request",
		Message:     "We received a request to reset your password. If you didn't request this, ignore this email.",
		ActionText:  "Reset Password",
		ActionLink:  m.cfg.PasswordResetURL(resetToken),
		RequestedAt: requestedAt.String(),
	})
}

func (m *Mailer) sendActionEmail(ctx context.Context, templateName, recipient, subject string, data ActionEmailData) error {
	return m.send(ctx, templateName, recipient, subject, data)
}
