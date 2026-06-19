package auth

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/sanitize"
	"bonfire-api/internal/user"
	"bonfire-api/internal/user_profile"
	"net/http"
)

type RegisterRequest struct {
	Email       string  `json:"email" validate:"required,email,max=255"`
	DisplayName *string `json:"display_name" validate:"omitempty,min=3,max=32"`
	Username    string  `json:"username" validate:"required,min=4,max=32,valid_username"`
	Password    string  `json:"password" validate:"required,min=12,max=128"`
}

type RegisterResponse struct {
	User        user.UserResponse                `json:"user"`
	UserProfile user_profile.UserProfileResponse `json:"user_profile"`
}

func (r *RegisterRequest) SanitizeRegisterRequest() {
	r.Email = sanitize.SanitizeEmail(r.Email)

	if r.DisplayName != nil {
		cleaned := sanitize.SanitizeText(*r.DisplayName)
		r.DisplayName = &cleaned
	}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) error {
	var req RegisterRequest

	// Decode request body
	if err := httpio.DecodeJSON(w, r, &req); err != nil {
		return err
	}

	// Sanitize request body
	req.SanitizeRegisterRequest()

	// Validate request body
	if err := h.val.ValidateStruct(&req); err != nil {
		return err
	}

	// Register user
	user, profile, err := h.service.Register(r.Context(), RegisterInput{
		Email:       req.Email,
		Username:    req.Username,
		DisplayName: req.DisplayName,
		Password:    req.Password,
	})
	if err != nil {
		return err
	}

	// Format response data
	data := RegisterResponse{
		User:        user,
		UserProfile: profile,
	}

	// Respond
	httpio.RespondCreated(w, data, "User successfully registered.")

	return nil
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email,max=255"`
	Password string `json:"password" validate:"required,min=12,max=128"`
}

type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func (r *LoginRequest) SanitizeLoginRequest() {
	r.Email = sanitize.SanitizeEmail(r.Email)
}

// Login handles user login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) error {
	var req LoginRequest

	// Decode request body
	if err := httpio.DecodeJSON(w, r, &req); err != nil {
		return err
	}

	// Sanitize request body
	req.SanitizeLoginRequest()

	// Validate request body
	if err := h.val.ValidateStruct(&req); err != nil {
		return err
	}

	// Extract client IP and User-Agent
	clientIP := r.RemoteAddr
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
	httpio.RespondJSON(w, http.StatusOK, map[string]string{
		"message":      LoginOkMsg,
		"access_token": tokens.AccessToken,
	})

	return nil
}

// RefreshToken handles requests to issue rotated access and refresh tokens
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) error {
	// 1. Extract the old refresh token from the HttpOnly cookie
	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		return apperr.New(apperr.CodeUnauthenticated, RefreshTokenMissingMsg)
	}

	// 2. Process the rotation request
	tokens, err := h.service.RefreshAccessToken(r.Context(), cookie.Value)
	if err != nil {
		return err
	}

	// 3. Set the NEW Refresh Token in the HttpOnly cookie (overwriting the old one)
	httpio.SetRefreshTokenCookie(w, tokens["refresh_token"])

	// 4. Respond with the fresh access token
	httpio.RespondJSON(w, http.StatusOK, map[string]string{
		"message":      RefreshTokenOkMsg,
		"access_token": tokens["access_token"],
	})

	return nil
}
