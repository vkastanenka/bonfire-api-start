package auth

import (
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/validator"
	"net/http"
	"time"
)

type AuthHandler struct {
	service *AuthService
	val     *validator.Validator
}

func NewAuthHandler(service *AuthService, val *validator.Validator) *AuthHandler {
	return &AuthHandler{service: service, val: val}
}

// Ping confirms the auth routes are available
func (h *AuthHandler) Ping(w http.ResponseWriter, r *http.Request) error {
	httpio.RespondJSON(w, http.StatusOK, map[string]string{
		"status": "healthy",
	})

	return nil
}

// Register handles user registration
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) error {
	var data RegisterData

	// Decode incoming JSON body into the struct
	if err := httpio.DecodeJSON(w, r, &data); err != nil {
		return err
	}

	// Validate request body
	if err := h.val.ValidateStruct(&data); err != nil {
		return err
	}

	// Register user
	if err := h.service.Register(r.Context(), data); err != nil {
		return err
	}

	// Respond
	httpio.RespondJSON(w, http.StatusCreated, map[string]string{
		"message": "User registered successfully!",
	})

	return nil
}

// Login handles user login
func (h *AuthHandler) LoginData(w http.ResponseWriter, r *http.Request) error {
	var data LoginData

	// Decode incoming JSON body into the struct
	if err := httpio.DecodeJSON(w, r, &data); err != nil {
		return err
	}

	// Validate request body
	if err := h.val.ValidateStruct(&data); err != nil {
		return err
	}

	// Extract client IP and User-Agent for session tracking
	clientIP := r.RemoteAddr // Note: Consider a helper to parse X-Forwarded-For if behind a proxy
	userAgent := r.UserAgent()

	// Login user, get tokens
	tokens, err := h.service.Login(r.Context(), data, userAgent, clientIP)
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
		Secure:   true, // Ensure this is true in production (requires HTTPS)
		SameSite: http.SameSiteStrictMode,
	})

	// Respond
	httpio.RespondJSON(w, http.StatusOK, map[string]string{
		"message":      "User login successful!",
		"access_token": tokens["access_token"],
	})

	return nil
}
