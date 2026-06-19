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
	req, err := httpio.BindJSON[RegisterRequest](w, r, h.val)
	if err != nil {
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

type RotateTokensResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// RotateTokens rotates access and refresh tokens
func (h *AuthHandler) RotateTokens(w http.ResponseWriter, r *http.Request) error {
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
