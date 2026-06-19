package auth

import (
	"bonfire-api/internal/httpio"
	"net/http"
)

// Login handles user login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) error {
	// Bind JSON
	req, err := httpio.BindJSON[LoginReq](w, r, h.val)
	if err != nil {
		return err
	}

	// Extract client IP and User-Agent
	clientIP := httpio.GetClientIP(r, false)
	userAgent := r.UserAgent()

	// Login user, get tokens
	tokens, err := h.service.Login(r.Context(), LoginParams{
		Email:     req.Email,
		Password:  req.Password,
		UserAgent: userAgent,
		ClientIP:  clientIP,
	})
	if err != nil {
		return err
	}

	// Set Refresh Token as an HttpOnly cookie
	httpio.SetRefreshTokenCookie(w, tokens.RefreshToken)

	// Respond
	httpio.RespondOK(w, LoginRes{AccessToken: tokens.AccessToken}, LoginOkMsg)

	return nil
}
