package auth

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/crypto"
	"bonfire-api/internal/outbox"
	"context"
	"log/slog"
	"time"
)

// TODO: Move to config
const (
	forgotPasswordTimingWindow = 35 * time.Millisecond
	forgotPasswordCooldown     = 1 * time.Minute
)

func (s *Service) ForgotPassword(ctx context.Context, rawEmail string) error {
	defer crypto.ConstantWindow(forgotPasswordTimingWindow)()

	onCooldown, err := s.cooldown.Get(ctx, "auth", "forgot-password", rawEmail)
	if err != nil {
		slog.ErrorContext(ctx, "forgot password cooldown lookup failed", "error", err, "email", rawEmail)
	} else if onCooldown {
		return nil
	}

	userRow, err := s.user.GetByEmail(ctx, rawEmail)
	if err != nil {
		if apperr.IsNotFound(err) {
			return nil
		}
		return err
	}

	t, _, err := s.token.GeneratePasswordReset(userRow.ID())
	if err != nil {
		return apperr.NewInternal(err)
	}

	persistCtx := context.WithoutCancel(ctx)

	_, err = s.outbox.Publish(persistCtx, outbox.PublishParams{
		Variant: EventForgotPassword,
		Payload: ForgotPasswordPayload{
			Email: userRow.Email().String(),
			Token: t,
		},
	})
	if err != nil {
		return err
	}

	if err := s.cooldown.Set(persistCtx, "auth", "forgot-password", userRow.Email().String(), forgotPasswordCooldown); err != nil {
		slog.WarnContext(persistCtx, "failed to set forgot password cooldown", "error", err, "email", userRow.Email().String())
	}

	return nil
}
