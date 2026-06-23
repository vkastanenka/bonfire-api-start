package auth

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/httpio"
	"net/http"
)

// Refresh rotates access and refresh tokens
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) error {
	// Check refresh token
	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		return apperr.New(apperr.CodeUnauthenticated, ErrMissingRefreshToken)
	}

	// Rotate access token
	tokens, err := h.service.Refresh(r.Context(), RefreshParams{
		RefreshToken: cookie.Value,
	})
	if err != nil {
		return err
	}

	// Repond with tokens
	httpio.SetRefreshTokenCookie(w, tokens.RefreshToken)
	httpio.RespondOK(w, r, RefreshRes{AccessToken: tokens.AccessToken}, RefreshTokenOk)

	return nil
}
