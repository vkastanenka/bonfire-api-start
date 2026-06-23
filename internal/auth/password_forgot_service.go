package auth

import (
	"context"
)

func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	// user, err := s.store.UserGetByEmail(ctx, email)
	// if err != nil {
	// 	if errors.Is(err, pgx.ErrNoRows) {
	// 		return nil
	// 	}
	// 	return apperr.New(apperr.CodeInternal, "System error", apperr.WithErr(err))
	// }

	// // Generate a short-lived token (15 mins) specifically for resetting
	// userID := uuid.UUID(user.ID.Bytes)
	// resetToken, err := s.tokenManager.GenerateJWT(userID, s.tokenConfig.PasswordResetSecret, 15*time.Minute)
	// if err != nil {
	// 	return err
	// }

	// // Create Outbox Event
	// jsonBytes, _ := json.Marshal(map[string]string{"email": email, "token": resetToken})
	// _, err = s.store.OutboxEventCreate(ctx, repository.OutboxEventCreateParams{
	// 	EventType: "user.forgot_password",
	// 	Payload:   jsonBytes,
	// })
	// return err
	return nil
}
