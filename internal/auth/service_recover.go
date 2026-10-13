package auth

import (
	"context"
	"log/slog"
	"time"

	"bonfire-api/internal/appctx"
	"bonfire-api/internal/outbox"
	"bonfire-api/internal/pkg/crypto"
	"bonfire-api/internal/pkg/errs"
	"bonfire-api/internal/pkg/utils"

	"github.com/google/uuid"
)

func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	blocked, ttl, err := s.shieldCache.GetCooldown(ctx, "auth", "forgot_password", email)
	if err != nil {
		slog.WarnContext(ctx, "failed to get forgot_password cooldown cache", slog.Any("error", err))
	} else if blocked {
		return ErrCooldownExceeded().Retry(ttl)
	}

	defer utils.ConstantWindow(ctx, forgotPasswordWindow)()

	u, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errs.IsNotFound(err) {
			return nil
		}
		return err
	}

	if err = u.EnsureActive(); err != nil {
		return nil
	}

	t, _, err := s.tokenProvider.GeneratePasswordReset(u.ID)
	if err != nil {
		return err
	}

	now := time.Now()

	event, err := outbox.New(
		EventPasswordResetRequested,
		outbox.NewMetadata(appctx.GetTraceID(ctx), &u.ID, nil, nil),
		newEventPasswordResetRequestedPayload(u.Email, u.Username, t, now),
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

	if err = s.shieldCache.SetCooldown(cacheCtx, "auth", "forgot_password", email); err != nil {
		slog.WarnContext(cacheCtx, "failed to set forgot_password cooldown", slog.Any("error", err))
	}

	return nil
}

type ResetPasswordParams struct {
	Token      string
	Password   string
	ClientMeta appctx.ClientMeta
}

type ResetPasswordResult struct {
	AccessToken           string
	RefreshToken          string
	RefreshTokenExpiresAt time.Time
}

func (s *Service) ResetPassword(ctx context.Context, p ResetPasswordParams) (ResetPasswordResult, error) {
	claims, err := s.tokenProvider.VerifyPasswordReset(p.Token)
	if err != nil {
		return ResetPasswordResult{}, err
	}

	if err = s.tokenCache.ConsumePasswordReset(ctx, claims); err != nil {
		return ResetPasswordResult{}, err
	}

	u, err := s.userRepo.Get(ctx, claims.UserID)
	if err != nil {
		if errs.IsNotFound(err) {
			return ResetPasswordResult{}, ErrTokenInvalid().Wrap(err)
		}
		return ResetPasswordResult{}, err
	}

	passwordHash, err := crypto.HashPassword(p.Password)
	if err != nil {
		return ResetPasswordResult{}, err
	}

	now := time.Now()

	sess, tokenPair, err := s.generateSession(u.ID, p.ClientMeta, now)
	if err != nil {
		return ResetPasswordResult{}, err
	}

	var revokedSessIDs []uuid.UUID

	txErr := s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		revokedSessIDs, err = s.sessionRepo.RevokeAll(txCtx, u.ID, now)
		if err != nil {
			return err
		}

		u, err = s.userRepo.UpdatePasswordHash(txCtx, u.ID, passwordHash, now)
		if err != nil {
			return err
		}

		if sess, err = s.sessionRepo.Create(txCtx, sess); err != nil {
			return err
		}

		if len(revokedSessIDs) > 0 {
			event, err := outbox.New(
				EventSessionRevokedAll,
				outbox.NewMetadata(appctx.GetTraceID(txCtx), &u.ID, &sess.ID, nil),
				newEventSessionRevokedAllPayload(revokedSessIDs),
				now,
			)
			if err != nil {
				return err
			}

			return s.outboxRepo.Create(txCtx, event)
		}

		return nil
	})
	if txErr != nil {
		return ResetPasswordResult{}, txErr
	}

	cacheCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()

	if err = s.cache.ResetPassword(cacheCtx, u, sess, revokedSessIDs); err != nil {
		slog.WarnContext(cacheCtx, "failed cache reset password pipeline",
			slog.String("user_id", u.ID.String()),
			slog.String("session_id", sess.ID.String()),
			slog.Any("error", err),
		)
	}

	return ResetPasswordResult{
		AccessToken:           tokenPair.Access,
		RefreshToken:          tokenPair.Refresh,
		RefreshTokenExpiresAt: tokenPair.RefreshExpiresAt,
	}, nil
}
