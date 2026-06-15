package email

import (
	"log"

	"bonfire-api/internal/config"
	"bonfire-api/internal/worker"
)

// NewMailer handles the conditional setup of the mailer infrastructure based on application configuration.
func NewMailer(cfg *config.Config) worker.Mailer {
	if cfg.ResendApiKey != "" {
		// Production / Staging Mode using Resend
		if cfg.EmailOverrideTo != "" {
			log.Printf("Email Engine: Resend initialized in SANDBOX mode (Overrides to: %s)", cfg.EmailOverrideTo)
		} else {
			log.Println("Email Engine: Resend initialized in PRODUCTION mode")
		}
		return NewResendMailer(cfg.ResendApiKey, cfg.EmailFromAddress, cfg.FrontendURL, cfg.EmailOverrideTo)
	}

	// Local Development Mode Fallback
	log.Println("Email Engine: Mock Mailer initialized (console output only)")
	return NewLogMockMailer()
}
