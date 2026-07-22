package auth

import (
	"context"
)

func (s *Service) VerifyEmail(ctx context.Context, tokenStr string) error {
	// claims, err := s.token.VerifyEmailVerify(tokenStr)
	// if err != nil {
	// 	// return apperr.NewTokenExpired(nil, apperr.CodeTokenExpired.Detail())
	// }

	// blacklistKey := cache.TokenBlacklistKey(claims.ID)
	// isBlacklisted, err := s.cache.Exists(ctx, blacklistKey)
	// if err != nil {
	// 	return apperr.NewInternal(err)
	// }
	// if isBlacklisted {
	// 	// return apperr.NewTokenExpired(nil, apperr.CodeTokenExpired.Detail())
	// }

	// persistCtx := context.WithoutCancel(ctx)

	// // _, err = s.user.MarkVerified(persistCtx, claims.UserID)
	// // if err != nil {
	// // 	return err
	// // }

	// remainingTTL := time.Until(claims.ExpiresAt.Time)
	// if remainingTTL > 0 {
	// 	s.cache.Set(persistCtx, blacklistKey, "true", remainingTTL)
	// }

	return nil
}
