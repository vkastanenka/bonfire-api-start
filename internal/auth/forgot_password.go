package auth

import (
	"bonfire-api/internal/apperr"
	"context"
	"log/slog"
	"time"
)

const (
	forgotPasswordTimingWindow = 35 * time.Millisecond
	forgotPasswordCooldown     = 1 * time.Minute
)

func (s *Service) ForgotPassword(ctx context.Context, rawEmail string) error {
	// defer crypto.ConstantWindow(forgotPasswordTimingWindow)()

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

	if err := outbox.EmitForgotPassword(persistCtx, s.store, outbox.ForgotPasswordPayload{
		Email: email,
		Token: t,
	}); err != nil {
		return err
	}

	// if err := s.cache.Set(persistCtx, cooldownKey, true, forgotPasswordCooldown); err != nil {
	// 	slog.WarnContext(persistCtx, "failed to set forgot password cooldown", "error", err, "email", email)
	// }

	return nil
}
