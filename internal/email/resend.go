package email

import (
	"context"
	"fmt"
	"log"

	"github.com/resend/resend-go/v3"
)

type ResendMailer struct {
	client      *resend.Client
	fromAddress string
	frontendURL string
	overrideTo  string // NEW: Catches emails in development
}

// NewResendMailer initializes the production email client.
// fromAddress should be a verified domain (e.g., "Bonfire <noreply@bonfire.app>")
// frontendURL is your React app's base URL (e.g., "http://localhost:5173" or "https://bonfire.app")
func NewResendMailer(apiKey, fromAddress, frontendURL, overrideTo string) *ResendMailer {
	return &ResendMailer{
		client:      resend.NewClient(apiKey),
		fromAddress: fromAddress,
		frontendURL: frontendURL,
		overrideTo:  overrideTo,
	}
}

func (m *ResendMailer) SendWelcomeEmail(ctx context.Context, emailAddress, username, token string) error {
	magicLink := fmt.Sprintf("%s/verify?token=%s", m.frontendURL, token)

	// 1. Determine the actual recipient
	recipient := emailAddress
	if m.overrideTo != "" {
		log.Printf("[RESEND] Sandbox mode: Redirecting email intended for %s to %s", emailAddress, m.overrideTo)
		recipient = m.overrideTo
	}

	// 2. Build the HTML payload (using a clean, Discord-esque layout)
	htmlBody := fmt.Sprintf(`
	<div style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px; background-color: #f9f9f9; border-radius: 8px;">
		<h2 style="color: #333;">Welcome to Bonfire, %s! 🔥</h2>
		<p style="color: #4f5660; font-size: 16px; line-height: 1.5;">
			We're excited to have you. Before you can start joining servers and chatting, you need to verify your email address.
		</p>
		<div style="margin: 30px 0; text-align: center;">
			<a href="%s" style="background-color: #5865F2; color: #ffffff; padding: 14px 28px; text-decoration: none; border-radius: 4px; font-weight: bold; display: inline-block;">
				Verify Email
			</a>
		</div>
		<p style="color: #72767d; font-size: 13px; margin-top: 40px; border-top: 1px solid #e3e5e8; padding-top: 20px;">
			If the button doesn't work, copy and paste this link into your browser:<br>
			<a href="%[2]s" style="color: #00a8fc; word-break: break-all;">%[2]s</a>
		</p>
	</div>
	`, username, magicLink)

	// 3. Configure the Resend request
	params := &resend.SendEmailRequest{
		From:    m.fromAddress,
		To:      []string{recipient}, // Use the intercepted address
		Subject: "Verify your Bonfire account",
		Html:    htmlBody,
	}

	// 4. Dispatch
	// Note: Resend's SDK doesn't natively take a context for the send method yet,
	// but we log the response ID for auditability in production.
	resp, err := m.client.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("failed to send welcome email: %w", err)
	}

	log.Printf("[RESEND] Successfully dispatched to %s (ID: %s)", recipient, resp.Id)
	return nil
}

func (m *ResendMailer) SendPasswordResetEmail(ctx context.Context, email, resetToken string) error {
	resetLink := fmt.Sprintf("%s/reset-password?token=%s", m.frontendURL, resetToken)

	// Build your HTML template here, similar to your Welcome Email
	htmlBody := fmt.Sprintf(`... Link: %s ...`, resetLink)

	params := &resend.SendEmailRequest{
		From:    m.fromAddress,
		To:      []string{email},
		Subject: "Reset your Bonfire password",
		Html:    htmlBody,
	}

	_, err := m.client.Emails.Send(params)
	return err
}
