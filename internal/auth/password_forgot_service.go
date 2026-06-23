package auth

import (
	"bonfire-api/internal/worker"
	"context"
)

func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	user, err := s.user.GetByEmail(ctx, email)
	if err != nil {
		return err
	}

	// Generate a short-lived token (15 mins) specifically for resetting
	resetToken, err := s.generatePasswordResetToken(user.ID)
	if err != nil {
		return err
	}

	// // Create Outbox Event
	err = worker.EmitEvent(ctx, qtx, worker.EventUserRegistered, worker.AuthForgotPasswordPayload{
		Email: email,
		Token: resetToken,
	})
	if err != nil {
		return err
	}

	return nil
}
