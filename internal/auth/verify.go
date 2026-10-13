package auth

import (
	"context"
	"log/slog"
	"time"

	"bonfire-api/internal/appctx"
	"bonfire-api/internal/outbox"
	"bonfire-api/internal/pkg/errs"
	"bonfire-api/internal/token"
	"bonfire-api/internal/user"
)

func (s *Service) VerifyEmail(ctx context.Context, token string) (*user.User, error) {
	claims, err := s.tokenProvider.VerifyEmailVerify(token)
	if err != nil {
		return nil, err
	}

	if err := s.tokenCache.ConsumeEmailVerify(ctx, claims); err != nil {
		return nil, err
	}

	now := time.Now()

	u, err := s.userRepo.Verify(ctx, claims.UserID, &now, now)
	if err != nil {
		return nil, err
	}

	if err := s.userCache.Delete(ctx, claims.UserID); err != nil {
		slog.WarnContext(ctx, "failed to invalidate user cache after email verification",
			slog.String("user_id", claims.UserID.String()),
			slog.Any("error", err),
		)
	}

	return u, nil
}

type ResendVerifyParams struct {
	Claims token.Claims
}

func (s *Service) ResendVerify(ctx context.Context, p ResendVerifyParams) error {
	u, err := s.cachedUserRepo.Get(ctx, p.Claims.UserID)
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
		outbox.NewMetadata(appctx.GetTraceID(ctx), &p.Claims.UserID, &p.Claims.SessionID, nil),
		newEventVerificationResentPayload(u.Email, u.Username, vToken, now),
		now,
	)
	if err != nil {
		return err
	}

	return s.outboxRepo.Create(ctx, event)
}
