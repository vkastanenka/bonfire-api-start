package auth

import (
	"bonfire-api/internal/httpio"
	"net/http"
)

// Login handles user login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) error {
	req, err := httpio.BindJSON[LoginRequest](w, r, h.val)
	if err != nil {
		return err
	}

	// Extract client IP and User-Agent
	clientIP := httpio.GetClientIP(r, false)
	userAgent := r.UserAgent()

	// Login user, get tokens
	tokens, err := h.service.Login(r.Context(), LoginInput{
		Email:    req.Email,
		Password: req.Password,
	}, userAgent, clientIP)
	if err != nil {
		return err
	}

	// Set Refresh Token as an HttpOnly cookie
	httpio.SetRefreshTokenCookie(w, tokens.RefreshToken)

	// Respond
	httpio.RespondOK(w, map[string]string{
		"access_token": tokens.AccessToken,
	}, LoginOkMsg)

	return nil
}
