package auth

import (
	"context"
	"fmt"
	"time"
)

const (
	forgotPasswordTimingWindow = 35 * time.Millisecond
	forgotPasswordCooldown     = 1 * time.Minute
)

func AuthCooldownForgotPasswordKey(email string) string {
	return fmt.Sprintf("auth:cooldown:forgot-password:{%s}", email)
}

func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	// defer crypto.ConstantWindow(forgotPasswordTimingWindow)()

	// cooldownKey := AuthCooldownForgotPasswordKey(email)
	// onCooldown, err := s.cache.Exists(ctx, cooldownKey)
	// if err != nil {
	// 	slog.ErrorContext(ctx, "forgot password cooldown lookup failed", "error", err, "email", email)
	// } else if onCooldown {
	// 	return nil
	// }

	// userRow, err := s.user.GetByEmail(ctx, email)
	// if err != nil {
	// 	return apperr.NewNotFound(err, "")
	// 	// if apperr.IsNotFound(err) {
	// 	// 	return nil
	// 	// }
	// 	// return err
	// }

	// t, _, err := s.token.GeneratePasswordReset(userRow.ID)
	// if err != nil {
	// 	return apperr.NewInternal(err, "")
	// }

	// persistCtx := context.WithoutCancel(ctx)

	// if err := outbox.EmitForgotPassword(persistCtx, s.store, outbox.ForgotPasswordPayload{
	// 	Email: email,
	// 	Token: t,
	// }); err != nil {
	// 	return err
	// }

	// if err := s.cache.Set(persistCtx, cooldownKey, true, forgotPasswordCooldown); err != nil {
	// 	slog.WarnContext(persistCtx, "failed to set forgot password cooldown", "error", err, "email", email)
	// }

	return nil
}
