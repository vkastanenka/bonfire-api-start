package auth

import (
	"bonfire-api/internal/httpio"
	"context"
	"net/http"
)

type VerifyEmailRequest struct {
	Token string `json:"token" validate:"required"`
}

// VerifyEmail handles incoming verification tokens sent from the frontend client.
func (h *Handler) VerifyEmail(w http.ResponseWriter, r *http.Request) error {
	var req VerifyEmailRequest

	if err := httpio.DecodeJSON(w, r, &req); err != nil {
		return err
	}

	if err := h.validator.ValidateStruct(&req); err != nil {
		return err
	}

	// Pass the token to the service method you just wrote
	if err := h.service.VerifyEmail(r.Context(), req.Token); err != nil {
		return err
	}

	httpio.RespondJSON(w, r, http.StatusOK, map[string]string{
		"message": "Your email address has been successfully verified!",
	})

	return nil
}

func (s *Service) VerifyEmail(ctx context.Context, tokenStr string) error {
	// // 1. Validate the stateless token structure
	// claims, err := s.tokenManager.VerifyJWT(tokenStr, s.tokenConfig.RefreshSecret)
	// if err != nil {
	// 	return apperr.New(apperr.CodeUnauthenticated, "The verification link is invalid or has expired.")
	// }

	// // 2. Perform safe, atomic bitwise alteration
	// err = s.store.UserMarkVerified(ctx, pgtype.UUID{Bytes: claims.UserID, Valid: true})
	// if err != nil {
	// 	return apperr.New(apperr.CodeInternal, "Failed to mark user as verified.", apperr.WithErr(err))
	// }

	return nil
}

type ResendVerificationEmailRequest struct {
	Email string `json:"email" validate:"required,email"`
}

func (h *Handler) ResendVerificationEmail(w http.ResponseWriter, r *http.Request) error {
	var data ResendVerificationEmailRequest

	if err := httpio.DecodeJSON(w, r, &data); err != nil {
		return err
	}

	if err := h.validator.ValidateStruct(&data); err != nil {
		return err
	}

	// Dispatch to service
	if err := h.service.ResendVerificationEmail(r.Context(), data.Email); err != nil {
		return err
	}

	// Return a generic 200 OK regardless of whether the email was found or not
	httpio.RespondJSON(w, r, http.StatusOK, map[string]string{
		"message": "If an unverified account exists with that email, a verification link has been sent.",
	})

	return nil
}

func (s *Service) ResendVerificationEmail(ctx context.Context, email string) error {
	return nil
	// // 1. Fetch the user
	// user, err := s.store.UserGetByEmail(ctx, email)
	// if err != nil {
	// 	// Security: If the user doesn't exist, return nil to prevent email enumeration
	// 	if errors.Is(err, pgx.ErrNoRows) {
	// 		return nil
	// 	}
	// 	return apperr.New(apperr.CodeInternal, "System error", apperr.WithErr(err))
	// }

	// // 2. Ensure they actually need verification
	// if user.VerifiedAt.Valid {
	// 	return apperr.New(apperr.CodeConflict, "This account is already verified.")
	// }

	// // 3. Enforce the Cooldown (e.g., 60 seconds)
	// if user.LastVerificationSentAt.Valid && time.Since(user.LastVerificationSentAt.Time) < 60*time.Second {
	// 	return apperr.New(apperr.CodeTooManyRequests, "Please wait a minute before requesting another verification email.")
	// }

	// // 4. Generate a fresh verification token
	// userID := uuid.UUID(user.ID.Bytes)
	// tokenStr, err := s.tokenManager.GenerateJWT(userID, s.tokenConfig.VerificationSecret, 24*time.Hour)
	// if err != nil {
	// 	return err
	// }

	// // 5. Execute Transaction: Update throttle timestamp AND queue the outbox event
	// return s.store.ExecTx(ctx, func(qtx *repository.Queries) error {
	// 	if err := qtx.UserUpdateLastVerificationSent(ctx, user.ID); err != nil {
	// 		return err
	// 	}

	// 	jsonBytes, _ := json.Marshal(map[string]string{
	// 		"email":    user.Email,
	// 		"username": user.Username,
	// 		"token":    tokenStr,
	// 	})

	// 	_, err = qtx.OutboxEventCreate(ctx, repository.OutboxEventCreateParams{
	// 		EventType: "user.verify_email", // New event type
	// 		Payload:   jsonBytes,
	// 	})
	// 	return err
	// })
}
