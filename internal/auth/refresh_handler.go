package auth

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/httpio"
	"net/http"
)

// Refresh rotates access and refresh tokens
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) error {
	// Check refresh token
	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		return apperr.New(apperr.CodeUnauthenticated, RefreshTokenMissingMsg)
	}

	// Rotate access token
	tokens, err := h.service.RotateTokens(r.Context(), cookie.Value)
	if err != nil {
		return err
	}

	// Set new refresh token in header
	httpio.SetRefreshTokenCookie(w, tokens.RefreshToken)

	// Respond with access token
	httpio.RespondOK(w, map[string]string{
		"access_token": tokens.AccessToken,
	}, RefreshTokenOkMsg)

	return nil
}
