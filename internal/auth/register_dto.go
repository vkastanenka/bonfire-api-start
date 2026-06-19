package auth

import (
	"bonfire-api/internal/sanitize"
	"bonfire-api/internal/user"
	"bonfire-api/internal/user_profile"
)

type RegisterReq struct {
	Email       string  `json:"email" validate:"required,email,max=255"`
	DisplayName *string `json:"display_name" validate:"omitempty,min=3,max=32"`
	Username    string  `json:"username" validate:"required,min=4,max=32,valid_username"`
	Password    string  `json:"password" validate:"required,min=12,max=128"`
}

func (r *RegisterReq) Sanitize() {
	r.Email = sanitize.SanitizeEmail(r.Email)

	if r.DisplayName != nil {
		cleaned := sanitize.SanitizeText(*r.DisplayName)
		r.DisplayName = &cleaned
	}
}

type RegisterParams struct {
	Email       string
	Username    string
	DisplayName *string
	Password    string
}

type RegisterResult struct {
	User        user.UserView                `json:"user"`
	UserProfile user_profile.UserProfileView `json:"user_profile"`
}
