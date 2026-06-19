package auth

import (
	"bonfire-api/internal/sanitize"
	"net/netip"
)

type LoginReq struct {
	Email    string `json:"email" validate:"required,email,max=255"`
	Password string `json:"password" validate:"required,min=12,max=128"`
}

func (r *LoginReq) Sanitize() {
	r.Email = sanitize.SanitizeEmail(r.Email)
}

type LoginParams struct {
	Email     string
	Password  string
	UserAgent string     `json:"user_agent"`
	ClientIP  netip.Addr `json:"client_ip"`
}

type LoginResult struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type LoginRes struct {
	AccessToken string `json:"access_token"`
}
