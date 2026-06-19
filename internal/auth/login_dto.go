package auth

import (
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/sanitize"
)

type LoginReq struct {
	Email    string `json:"email" validate:"required,email,max=255"`
	Password string `json:"password" validate:"required,min=12,max=128"`
}

func (r *LoginReq) Sanitize() {
	r.Email = sanitize.SanitizeEmail(r.Email)
}

type LoginParams struct {
	Email    string
	Password string
	Meta     httpio.ClientMeta
}

type LoginResult struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type LoginRes struct {
	AccessToken string `json:"access_token"`
}
