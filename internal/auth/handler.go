package auth

import (
	"bonfire-api/internal/apperr"
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

// VerifyEmail handles incoming verification tokens sent from the frontend client.
func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) error {
	var req VerifyEmailData

	if err := httpio.DecodeJSON(w, r, &req); err != nil {
		return err
	}

	if err := h.val.ValidateStruct(&req); err != nil {
		return err
	}

	// Pass the token to the service method you just wrote
	if err := h.service.VerifyEmail(r.Context(), req.Token); err != nil {
		return err
	}

	httpio.RespondJSON(w, http.StatusOK, map[string]string{
		"message": "Your email address has been successfully verified!",
	})

	return nil
}

// Login handles user login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) error {
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

// RefreshToken handles requests to issue rotated access and refresh tokens
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) error {
	// 1. Extract the old refresh token from the HttpOnly cookie
	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		return apperr.NewUnauthenticated("Missing refresh token. Please log in.")
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

// ForgotPassword initiates the password reset flow
func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) error {
	var data ForgotPasswordData

	if err := httpio.DecodeJSON(w, r, &data); err != nil {
		return err
	}

	if err := h.val.ValidateStruct(&data); err != nil {
		return err
	}

	// Pass to service. We don't care if the user doesn't exist,
	// the service handles the logic silently to prevent enumeration.
	if err := h.service.ForgotPassword(r.Context(), data.Email); err != nil {
		return err
	}

	// Success response is generic to prevent email enumeration
	httpio.RespondJSON(w, http.StatusOK, map[string]string{
		"message": "If an account exists with this email, a password reset link has been sent.",
	})

	return nil
}

// ResetPassword finalizes the password change
func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) error {
	var data ResetPasswordData

	if err := httpio.DecodeJSON(w, r, &data); err != nil {
		return err
	}

	if err := h.val.ValidateStruct(&data); err != nil {
		return err
	}

	if err := h.service.ResetPassword(r.Context(), data.Token, data.NewPassword); err != nil {
		return err
	}

	httpio.RespondJSON(w, http.StatusOK, map[string]string{
		"message": "Your password has been reset successfully. You may now log in.",
	})

	return nil
}

// ----------------------------------------------------
// PROTECTED ROUTES (Valid Access Token required) example
// ----------------------------------------------------
// func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
//     // 1. Pull the user ID out of the context
//     userID, err := auth.GetUserIDFromContext(r.Context())
//     if err != nil {
//         httpio.RespondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Context failure"})
//         return
//     }

//     // 2. Fetch data specifically for this user
//     // userProfile, err := h.service.GetUserProfileByID(r.Context(), userID)
//     // ...
// }
