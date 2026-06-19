package auth

import "bonfire-api/internal/sanitize"

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

// LoginInput
type LoginInput struct {
	Email    string
	Password string
}
