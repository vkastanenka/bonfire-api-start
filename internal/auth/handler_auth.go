package auth

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/httpio"
	"net/http"
	"time"
)

type RegisterRequest struct {
	Email       string  `json:"email" validate:"required,email,max=255"`
	DisplayName *string `json:"displayName" validate:"omitempty,min=3,max=32"`
	Username    string  `json:"username" validate:"required,alphanum,min=8,max=32"`
	Password    string  `json:"password" validate:"required,min=8,max=100"`
}

// Register godoc
// @Summary      Register a new user
// @Description  Create a new account
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body auth.RegisterRequest true "User Registration"
// @Success      201 {object} httpio.Response
// @Failure      400 {object} apperr.AppError
// @Router       /auth/register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) error {
	var data RegisterRequest

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

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email,max=255"`
	Password string `json:"password" validate:"required,min=8,max=100"`
}

// Login handles user login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) error {
	var data LoginRequest

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

// RefreshToken handles requests to issue rotated access and refresh tokens
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) error {
	// 1. Extract the old refresh token from the HttpOnly cookie
	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		return apperr.New(apperr.CodeUnauthenticated, "Missing refresh token. Please log in.")
	}

	// 2. Process the rotation request
	tokens, err := h.service.RefreshAccessToken(r.Context(), cookie.Value)
	if err != nil {
		return err
	}

	// 3. Set the NEW Refresh Token in the HttpOnly cookie (overwriting the old one)
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    tokens["refresh_token"],
		Path:     "/",
		Expires:  time.Now().Add(7 * 24 * time.Hour), // Matches the service duration
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})

	// 4. Respond with the fresh access token
	httpio.RespondJSON(w, http.StatusOK, map[string]string{
		"access_token": tokens["access_token"],
	})

	return nil
}
