package auth

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/httpio"
	"context"
	"net/http"
)

// --- VERIFY EMAIL CONSTANTS ---

// Messages
const (
	MsgResendVerificationEmailSuccess = "resend_verification_email_success"
)

// Errors
const (
	ErrAccountVerified = "This account is already verified."
)

// --- VERIFY EMAIL TYPES ---

type ResendVerificationEmailReq struct {
	Email string `json:"email" validate:"required,email"`
}

func (h *Handler) ResendVerificationEmail(w http.ResponseWriter, r *http.Request) error {
	// Get JSON
	req, err := httpio.BindJSON[ResendVerificationEmailReq](w, r, h.validator)
	if err != nil {
		return err
	}

	// Call service
	if err := h.service.ResendVerificationEmail(r.Context(), req.Email); err != nil {
		return err
	}

	// Return a generic 200 OK regardless of whether the email was found or not
	httpio.RespondOK(w, r, struct{}{}, MsgResendVerificationEmailSuccess)

	return nil
}

func (s *Service) ResendVerificationEmail(ctx context.Context, email string) error {
	// Get user
	user, err := s.user.GetByEmail(ctx, email)
	if err != nil {
		if apperr.Is(err, apperr.CodeNotFound) {
			return nil
		}
		return err
	}

	// Ensure they actually need verification
	if user.VerifiedAt != nil {
		return apperr.New(apperr.CodeConflict, ErrAccountVerified)
	}

	// 3. Enforce the Cooldown (e.g., 60 seconds)
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
	return nil
}
