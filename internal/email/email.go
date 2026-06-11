package email

import (
	"context"
	"log"
)

type LogMockMailer struct{}

func NewLogMockMailer() *LogMockMailer {
	return &LogMockMailer{}
}

func (m *LogMockMailer) SendWelcomeEmail(ctx context.Context, email string, username string, token string) error {
	log.Printf("[MOCK EMAIL] >>> Successfully dispatched welcome packet to %s (Username: %s, Token: %s)!", email, username, token)
	return nil
}
