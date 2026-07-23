package auth

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"bonfire-api/internal/crypto"
	"bonfire-api/internal/errs"
	"bonfire-api/internal/outbox"
	"bonfire-api/internal/sanitize"
	"bonfire-api/internal/user"
)

// TODO: Move to config
const (
	forgotPasswordTimingWindow = 35 * time.Millisecond
	forgotPasswordCooldown     = 1 * time.Minute
)

func (s *Service) ForgotPassword(ctx context.Context, rawEmail string) error {
	defer crypto.ConstantWindow(forgotPasswordTimingWindow)()

	email, err := user.NewEmail(sanitize.Email(rawEmail))
	if err != nil || !email.IsValid() {
		return errs.InvalidArgument("Invalid email address.").
			FieldViolation("email", "Must be a valid email address.", "INVALID_EMAIL").
			Wrap(errors.New("invalid email address"))
	}

	onCooldown, err := s.shield.GetCooldown(ctx, "auth", "forgot-password", email.String())
	if err != nil {
		slog.ErrorContext(ctx, "forgot password cooldown lookup failed", "error", err, "email", email.String())
	} else if onCooldown {
		// Silent pass to prevent enumeration
		return nil
	}

	persistCtx := context.WithoutCancel(ctx)

	// Always set cooldown regardless of user existence to prevent account enumeration
	if err := s.shield.SetCooldown(persistCtx, "auth", "forgot-password", email.String(), forgotPasswordCooldown); err != nil {
		slog.WarnContext(persistCtx, "failed to set forgot password cooldown", "error", err, "email", email.String())
	}

	userRow, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if errs.IsNotFound(err) {
			return nil
		}
		return err
	}

	t, _, err := s.tokens.GeneratePasswordReset(userRow.ID())
	if err != nil {
		return errs.Internal("failed to generate password reset token").Wrap(err)
	}

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

	return nil
}
