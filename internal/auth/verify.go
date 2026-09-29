package auth

import (
	"context"
	"log/slog"
	"time"

	"bonfire-api/internal/appctx"
	"bonfire-api/internal/outbox"
	"bonfire-api/internal/pkg/errs"
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

func (s *Service) ResendVerify(ctx context.Context) error {
	ctxClaims, err := appctx.GetClaims(ctx)
	if err != nil {
		return err
	}

	u, err := s.cachedUserRepo.Get(ctx, ctxClaims.UserID)
	if err != nil {
		if errs.IsNotFound(err) {
			return nil
		}
		return err
	}

	verifyToken, _, err := s.tokenProvider.GenerateEmailVerify(u.ID)
	if err != nil {
		return err
	}

	now := time.Now()

	payload := EventVerificationResentPayload{
		Email:       u.Email,
		Username:    u.Username,
		Token:       verifyToken,
		RequestedAt: now,
	}

	event, err := outbox.New(
		ctxClaims.UserID,
		ctxClaims.SessionID,
		appctx.GetTraceID(ctx),
		nil,
		EventVerificationResent,
		payload,
		now,
	)
	if err != nil {
		return err
	}

	return s.outboxRepo.Create(ctx, event)
}
