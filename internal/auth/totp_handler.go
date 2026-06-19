package auth

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/httpio"
	"net/http"
	"time"
)

// GenerateTOTP creates a new 2FA setup intent for the logged-in user.
func (h *AuthHandler) GenerateTOTP(w http.ResponseWriter, r *http.Request) error {
	// 1. Pull the user ID out of the context
	userID, err := GetUserIDFromContext(r.Context())
	if err != nil {
		return apperr.New(apperr.CodeUnauthenticated, "Missing user identity in context.")
	}

	// 2. Delegate directly to the auth service passing the userID
	secret, url, err := h.service.GenerateTOTP(r.Context(), userID)
	if err != nil {
		return err
	}

	httpio.RespondJSON(w, http.StatusOK, map[string]string{
		"secret": secret,
		"url":    url,
	})

	return nil
}

type EnableTOTPRequest struct {
	Secret string `json:"secret" validate:"required"`
	Code   string `json:"code" validate:"required,len=6,numeric"`
}

// EnableTOTP validates the user's first 6-digit code and permanently activates 2FA.
func (h *AuthHandler) EnableTOTP(w http.ResponseWriter, r *http.Request) error {
	var data EnableTOTPRequest

	if err := httpio.DecodeJSON(w, r, &data); err != nil {
		return err
	}

	if err := h.val.ValidateStruct(&data); err != nil {
		return err
	}

	// 1. Pull the user ID out of the context (requires access token)
	userID, err := GetUserIDFromContext(r.Context())
	if err != nil {
		return apperr.New(apperr.CodeUnauthenticated, "Missing user identity in context.")
	}

	// 2. Pass to service for cryptographic verification and database update
	if err := h.service.EnableTOTP(r.Context(), userID, data.Secret, data.Code); err != nil {
		return err
	}

	httpio.RespondJSON(w, http.StatusOK, map[string]string{
		"message": "Two-factor authentication has been successfully enabled.",
	})

	return nil
}

type VerifyLogin2FARequest struct {
	MFAToken string `json:"mfaToken" validate:"required"`
	Code     string `json:"code" validate:"required,len=6,numeric"`
}

// VerifyLogin2FA handles the second step of the login flow if the user has 2FA enabled.
func (h *AuthHandler) VerifyLogin2FA(w http.ResponseWriter, r *http.Request) error {
	var data VerifyLogin2FARequest

	if err := httpio.DecodeJSON(w, r, &data); err != nil {
		return err
	}

	if err := h.val.ValidateStruct(&data); err != nil {
		return err
	}

	// Extract client IP and User-Agent for session tracking
	clientIP := r.RemoteAddr
	userAgent := r.UserAgent()

	// Validate the MFA token and the 6-digit code to finalize the login
	tokens, err := h.service.VerifyLogin2FA(r.Context(), data.MFAToken, data.Code, userAgent, clientIP)
	if err != nil {
		return err
	}

	// Set Refresh Token as an HttpOnly cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    tokens["refresh_token"],
		Path:     "/",
		Expires:  time.Now().Add(7 * 24 * time.Hour),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})

	// Respond with the fresh access token
	httpio.RespondJSON(w, http.StatusOK, map[string]string{
		"message":      "2FA login successful!",
		"access_token": tokens["access_token"],
	})

	return nil
}
