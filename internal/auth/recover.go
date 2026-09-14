package auth

import (
	"context"
	"time"

	"bonfire-api/internal/crypto"
	"bonfire-api/internal/errs"
	"bonfire-api/internal/httpio"
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
		Email: userRow.Email,
		Token: t,
	}

	return s.outboxRepo.Publish(ctx, EventForgotPassword, payload, now)
}

type ResetPasswordParams struct {
	Token      string
	Password   string
	ClientMeta httpio.ClientMeta
}

type ResetPasswordResult struct {
	AccessToken           string
	RefreshToken          string
	RefreshTokenExpiresAt time.Time
}

func (s *Service) ResetPassword(ctx context.Context, p ResetPasswordParams) (ResetPasswordResult, error) {
	claims, err := s.tokenProvider.VerifyPasswordReset(p.Token)
	if err != nil {
		return ResetPasswordResult{}, ErrResetTokenInvalid(err)
	}

	if err := s.tokenCache.ConsumePasswordResetToken(ctx, claims); err != nil {
		return ResetPasswordResult{}, err
	}

	u, err := s.userRepo.Get(ctx, claims.UserID)
	if err != nil {
		if errs.IsNotFound(err) {
			return ResetPasswordResult{}, ErrResetTokenUserNotFound(err)
		}
		return ResetPasswordResult{}, err
	}

	passwordHash, err := crypto.HashPassword(p.Password)
	if err != nil {
		return ResetPasswordResult{}, err
	}

	now := time.Now()

	newSession, tokenPair, err := s.generateSession(u, p.ClientMeta, now)
	if err != nil {
		return ResetPasswordResult{}, err
	}

	var revokedSessionIDs []uuid.UUID
	var updatedUser *user.User

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

		if _, err := s.sessionRepo.Create(txCtx, newSession); err != nil {
			return err
		}

		if len(revokedSessionIDs) > 0 {
			revokedSessionIDStrings := make([]string, len(revokedSessionIDs))
			for i, id := range revokedSessionIDs {
				revokedSessionIDStrings[i] = id.String()
			}

			revokePayload := session.EventRevokeAllPayload{
				UserID:     u.ID,
				SessionIDs: revokedSessionIDs,
				RevokedAt:  now,
			}

			if err := s.outboxRepo.Publish(txCtx, session.EventRevokeAll, revokePayload, now); err != nil {
				return err
			}
		}

		return nil
	})

	if txErr != nil {
		return ResetPasswordResult{}, txErr
	}

	if len(revokedSessionIDs) > 0 {
		_ = s.sessionCache.DeleteBatch(ctx, revokedSessionIDs)
	}

	_ = s.sessionCache.Set(ctx, newSession)
	_ = s.userCache.Set(ctx, updatedUser)

	return ResetPasswordResult{
		AccessToken:           tokenPair.Access,
		RefreshToken:          tokenPair.Refresh,
		RefreshTokenExpiresAt: tokenPair.RefreshExpiresAt,
	}, nil
}
