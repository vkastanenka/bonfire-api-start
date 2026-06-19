package auth

import (
	"bonfire-api/internal/sanitize"
	"bonfire-api/internal/user"
	"bonfire-api/internal/user_profile"
)

// RegisterInput
type RegisterInput struct {
	Email       string
	Username    string
	DisplayName *string
	Password    string
}

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
