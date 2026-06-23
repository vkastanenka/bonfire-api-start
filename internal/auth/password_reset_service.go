package auth

import (
	"context"
)

func (s *Service) ResetPassword(ctx context.Context, tokenStr string, newPassword string) error {
	// // Verify the token using the PasswordResetSecret
	// claims, err := s.tokenManager.VerifyJWT(tokenStr, s.tokenConfig.PasswordResetSecret)
	// if err != nil {
	// 	return apperr.New(apperr.CodeUnauthenticated, "Invalid or expired reset token.")
	// }

	// // Hash the new password
	// hashedPasswordBytes, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	// if err != nil {
	// 	return apperr.New(apperr.CodeInternal, "Failed to hash password", apperr.WithErr(err))
	// }

	// // Execute update
	// err = s.store.UserUpdatePassword(ctx, repository.UserUpdatePasswordParams{
	// 	ID:           pgtype.UUID{Bytes: claims.UserID, Valid: true},
	// 	PasswordHash: string(hashedPasswordBytes),
	// })
	// if err != nil {
	// 	return apperr.New(apperr.CodeInternal, "Failed to update password", apperr.WithErr(err))
	// }

	return nil
}
