package auth

import (
	"context"
	"log/slog"
	"time"

	"bonfire-api/internal/appctx"
	"bonfire-api/internal/outbox"
	"bonfire-api/internal/pkg/errs"
	"bonfire-api/internal/user"

	"github.com/google/uuid"
)

func (s *Service) VerifyEmail(ctx context.Context, token string) (*user.User, error) {
	claims, err := s.tokenProvider.VerifyEmailVerify(token)
	if err != nil {
		return nil, err
	}

	if err = s.tokenCache.ConsumeEmailVerify(ctx, claims); err != nil {
		return nil, err
	}

	now := time.Now()

	u, err := s.userRepo.Verify(ctx, claims.UserID, &now, now)
	if err != nil {
		return nil, err
	}

	cacheCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()

	if err := s.userCache.Delete(cacheCtx, claims.UserID); err != nil {
		slog.WarnContext(cacheCtx, "failed to invalidate user cache after email verification",
			slog.String("user_id", claims.UserID.String()),
			slog.Any("error", err),
		)
	}

	return u, nil
}

type ResendVerifyParams struct {
	UserID    uuid.UUID
	SessionID uuid.UUID
}

func (s *Service) ResendVerify(ctx context.Context, p ResendVerifyParams) error {
	blocked, ttl, err := s.shieldCache.GetCooldown(ctx, "auth", "resend_verify", p.UserID.String())
	if err != nil {
		slog.WarnContext(ctx, "failed to get resend_verify cooldown cache", slog.Any("error", err))
	} else if blocked {
		return ErrCooldownExceeded().Retry(ttl)
	}

	u, err := s.cachedUserRepo.Get(ctx, p.UserID)
	if err != nil {
		if errs.IsNotFound(err) {
			return nil
		}
		return err
	}

	vToken, _, err := s.tokenProvider.GenerateEmailVerify(u.ID)
	if err != nil {
		return err
	}

	now := time.Now()

	event, err := outbox.New(
		EventVerificationResent,
		outbox.NewMetadata(appctx.GetTraceID(ctx), &p.UserID, &p.SessionID, nil),
		newEventVerificationResentPayload(u.Email, u.Username, vToken, now),
		now,
	)
	if err != nil {
		return err
	}

	err = s.outboxRepo.Create(ctx, event)
	if err != nil {
		return err
	}

	cacheCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()

	if err = s.shieldCache.SetCooldown(cacheCtx, "auth", "resend_verify", p.UserID.String()); err != nil {
		slog.WarnContext(cacheCtx, "failed to set resend_verify cooldown", slog.Any("error", err))
	}

	return nil
}
