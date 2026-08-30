package auth

import (
	"context"
	"time"

	"bonfire-api/internal/crypto"
	"bonfire-api/internal/errs"
	"bonfire-api/internal/fields"
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/user"
)

const (
	forgotPasswordTimingWindow = 35 * time.Millisecond
)

func (s *Service) ForgotPassword(ctx context.Context, rawEmail string) error {
	defer crypto.ConstantWindow(forgotPasswordTimingWindow)()

	// email, err := user.ParseRequiredEmail("email", rawEmail)
	// if err != nil || !email.IsValid() {
	// 	return nil
	// }

	// userRow, err := s.userRepo.GetByEmail(ctx, email)
	// if err != nil {
	// 	if errs.IsNotFound(err) {
	// 		return nil
	// 	}
	// 	return err
	// }

	// t, _, err := s.tokenProvider.GeneratePasswordReset(userRow.ID())
	// if err != nil {
	// 	return err
	// }

	// return s.outboxRepo.Publish(ctx, EventForgotPassword, ForgotPasswordPayload{
	// 	Email: userRow.Email().String(),
	// 	Token: t,
	// })
	return nil
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

// ResetPassword verifies the reset token, updates the user's password, invalidates
// all active sessions, and logs the user in with a fresh session pair.
func (s *Service) ResetPassword(ctx context.Context, p ResetPasswordParams) (ResetPasswordResult, error) {
	token, err := fields.ParseRequiredToken("token", p.Token)
	if err != nil {
		return ResetPasswordResult{}, err
	}

	password, err := user.ParseRequiredPassword("password", p.Password)
	if err != nil {
		return ResetPasswordResult{}, err
	}

	claims, err := s.tokenProvider.VerifyPasswordReset(token.String())
	if err != nil {
		return ResetPasswordResult{}, ErrResetTokenInvalid(err)
	}

	u, err := s.userRepo.Get(ctx, claims.UserID)
	if err != nil {
		if errs.IsNotFound(err) {
			return ResetPasswordResult{}, ErrResetTokenUserNotFound(err)
		}
		return ResetPasswordResult{}, err
	}

	rawPasswordHash, err := crypto.HashPassword(password.String())
	if err != nil {
		return ResetPasswordResult{}, err
	}

	passwordHash := user.NewPasswordHash(rawPasswordHash)

	now := fields.Now()

	newSession, tokenPair, err := s.generateSession(u, p.ClientMeta, now)

	txErr := s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		if err := s.sessionRepo.RevokeAll(txCtx, u.ID(), now); err != nil {
			return err
		}

		if _, err := s.userRepo.UpdatePasswordHash(txCtx, u.ID(), passwordHash, now); err != nil {
			return err
		}

		if _, err := s.sessionRepo.Create(txCtx, newSession); err != nil {
			return err
		}

		return nil
	})

	if txErr != nil {
		return ResetPasswordResult{}, txErr
	}

	return ResetPasswordResult{
		AccessToken:           tokenPair.Access,
		RefreshToken:          tokenPair.Refresh,
		RefreshTokenExpiresAt: tokenPair.RefreshExpiresAt,
	}, nil
}
