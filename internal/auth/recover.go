package auth

import (
	"context"
	"log/slog"
	"time"

	"bonfire-api/internal/appctx"
	"bonfire-api/internal/outbox"
	"bonfire-api/internal/pkg/crypto"
	"bonfire-api/internal/pkg/errs"
	"bonfire-api/internal/session"
	"bonfire-api/internal/user"

	"github.com/google/uuid"
)

const (
	forgotPasswordTimingWindow = 35 * time.Millisecond
)

func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	defer crypto.ConstantWindow(ctx, forgotPasswordTimingWindow)()

	userRow, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errs.IsNotFound(err) {
			return nil
		}
		return err
	}

	if err := userRow.EnsureActive(); err != nil {
		return nil
	}

	t, _, err := s.tokenProvider.GeneratePasswordReset(userRow.ID)
	if err != nil {
		return err
	}

	now := time.Now()

	payload := EventForgotPasswordPayload{
		Email:     userRow.Email,
		Token:     t,
		CreatedAt: now,
	}

	event, err := outbox.New(
		uuid.Nil,
		uuid.Nil,
		appctx.GetTraceID(ctx),
		nil,
		EventForgotPassword,
		payload,
		now,
	)
	if err != nil {
		return err
	}

	return s.outboxRepo.Create(ctx, event)
}

type ResetPasswordParams struct {
	Token    string
	Password string
}

type ResetPasswordResult struct {
	AccessToken           string
	RefreshToken          string
	RefreshTokenExpiresAt time.Time
}

func (s *Service) ResetPassword(ctx context.Context, p ResetPasswordParams) (ResetPasswordResult, error) {
	claims, err := s.tokenProvider.VerifyPasswordReset(p.Token)
	if err != nil {
		return ResetPasswordResult{}, ErrResetTokenInvalid().Wrap(err)
	}

	if err := s.tokenCache.ConsumePasswordReset(ctx, claims); err != nil {
		return ResetPasswordResult{}, err
	}

	u, err := s.userRepo.Get(ctx, claims.UserID)
	if err != nil {
		if errs.IsNotFound(err) {
			return ResetPasswordResult{}, ErrResetTokenUserNotFound().Wrap(err)
		}
		return ResetPasswordResult{}, err
	}

	passwordHash, err := crypto.HashPassword(p.Password)
	if err != nil {
		return ResetPasswordResult{}, err
	}

	now := time.Now()

	newSession, tokenPair, err := s.generateSession(ctx, u, now)
	if err != nil {
		return ResetPasswordResult{}, err
	}

	var revokedSessionIDs []uuid.UUID
	var updatedUser *user.User
	var dbSession *session.Session

	txErr := s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		var err error
		revokedSessionIDs, err = s.sessionRepo.RevokeAll(txCtx, u.ID, now)
		if err != nil {
			return err
		}

		updatedUser, err = s.userRepo.UpdatePasswordHash(txCtx, u.ID, passwordHash, now)
		if err != nil {
			return err
		}

		if dbSession, err = s.sessionRepo.Create(txCtx, newSession); err != nil {
			return err
		}

		if len(revokedSessionIDs) > 0 {
			payload := session.EventRevokedAllPayload{
				UserID:     u.ID,
				SessionIDs: revokedSessionIDs,
				RevokedAt:  now,
			}

			event, err := outbox.New(
				updatedUser.ID,
				uuid.Nil,
				appctx.GetTraceID(txCtx),
				nil,
				session.EventRevokedAll,
				payload,
				now,
			)
			if err != nil {
				return err
			}

			if err := s.outboxRepo.Create(txCtx, event); err != nil {
				return err
			}
		}

		return nil
	})

	if txErr != nil {
		return ResetPasswordResult{}, txErr
	}

	if len(revokedSessionIDs) > 0 {
		if err := s.sessionCache.DeleteBatch(ctx, revokedSessionIDs); err != nil {
			slog.WarnContext(ctx, "failed to invalidate revoked sessions in cache after password reset",
				slog.String("user_id", u.ID.String()),
				slog.Int("count", len(revokedSessionIDs)),
				slog.Any("error", err),
			)
		}
	}

	if err := s.sessionCache.Set(ctx, dbSession); err != nil {
		slog.WarnContext(ctx, "failed to seed new session into cache after password reset",
			slog.String("user_id", u.ID.String()),
			slog.String("session_id", dbSession.ID.String()),
			slog.Any("error", err),
		)
	}

	if err := s.userCache.Set(ctx, updatedUser); err != nil {
		slog.WarnContext(ctx, "failed to update user in cache after password reset",
			slog.String("user_id", u.ID.String()),
			slog.Any("error", err),
		)
	}

	return ResetPasswordResult{
		AccessToken:           tokenPair.Access,
		RefreshToken:          tokenPair.Refresh,
		RefreshTokenExpiresAt: tokenPair.RefreshExpiresAt,
	}, nil
}
