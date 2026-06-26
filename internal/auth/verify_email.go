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
	MsgVerifyEmailSuccess = "verify_email_success"
)

// Errors
const (
	ErrInvalidRefreshToken = "The refresh token is invalid or expired."
)

// --- VERIFY EMAIL TYPES ---

type VerifyEmailReq struct {
	Token string `json:"token" validate:"required"`
}

// --- VERIFY EMAIL HANDLER ---

// VerifyEmail handles incoming verification tokens sent from the frontend client.
func (h *Handler) VerifyEmail(w http.ResponseWriter, r *http.Request) error {
	// Get JSON
	req, err := httpio.BindJSON[VerifyEmailReq](w, r, h.validator)
	if err != nil {
		return err
	}

	// Verify email
	if err := h.service.VerifyEmail(r.Context(), req.Token); err != nil {
		return err
	}

	// Respond
	httpio.RespondOK(w, r, struct{}{}, MsgVerifyEmailSuccess)

	return nil
}

// --- VERIFY EMAIL SERVICE ---

func (s *Service) VerifyEmail(ctx context.Context, tokenStr string) error {
	// Verify refresh token
	claims, err := s.token.VerifyVerification(tokenStr)
	if err != nil {
		return apperr.New(apperr.CodeUnauthorized, ErrInvalidRefreshToken, apperr.WithErr(err))
	}

	// // 2. Enforce Single-Use via a Cache Blocklist (Prevents replay attacks)
	// // We use the token's JTI (or a cryptographic hash of the signature if stateless)
	// usedTokenKey := cache.UsedVerificationTokenKey(claims.TokenID)

	// isUsed, err := s.cache.Exists(ctx, usedTokenKey)
	// if err != nil {
	// 	// Fail-closed on security check errors to protect the state transition
	// 	return apperr.NewInternal(err)
	// }
	// if isUsed {
	// 	return apperr.New(apperr.CodeUnauthorized, ErrInvalidVerifyToken)
	// }

	// 3. Create non-cancelable context for persistent side effects
	persistCtx := context.WithoutCancel(ctx)

	// Mark user verification
	_, err = s.user.MarkVerified(persistCtx, claims.UserID)
	if err != nil {
		return err
	}

	// // 5. Consume the token by caching its ID until its original expiration window passes
	// timeUntilExpiry := time.Until(claims.ExpiresAt)
	// if timeUntilExpiry > 0 {
	// 	if cacheErr := s.cache.Set(persistCtx, usedTokenKey, true, timeUntilExpiry); cacheErr != nil {
	// 		// Log but don't break the user experience; your database update succeeded
	// 		slog.ErrorContext(ctx, "failed to track consumed email verification token", "error", cacheErr, "token_id", claims.TokenID)
	// 	}
	// }

	return nil
}
