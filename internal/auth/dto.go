package dto

import (
	"context"
	"strings"

	"github.com/go-playground/validator/v10"
)

type RegisterRequest struct {
	Email       string `json:"email" validate:"required,email,max=255"`
	DisplayName string `json:"displayName" validate:"required,min=1,max=32"`
	Username    string `json:"username" validate:"required,alphanum,min=8,max=32"`
	Password    string `json:"password" validate:"required,secure_password,max=72"`
}

// Sanitize cleans up the input before validation runs
func (r *RegisterRequest) Sanitize() {
	r.Email = strings.TrimSpace(strings.ToLower(r.Email))
	r.Username = strings.TrimSpace(r.Username)
	r.DisplayName = strings.TrimSpace(r.DisplayName)
}

// Valid fulfills the Validatable interface and returns custom user-friendly errors
func (r *RegisterRequest) Valid(ctx context.Context) map[string]string {
	r.Sanitize()

	err := validate.StructCtx(ctx, r)
	if err == nil {
		return nil
	}

	problems := make(map[string]string)
	for _, err := range err.(validator.ValidationErrors) {
		// Map structural field names to their JSON counterparts for the client
		field := err.Field() 
		switch field {
		case "Email":
			problems["email"] = "Must be a valid email address under 255 characters."
		case "DisplayName":
			problems["displayName"] = "Display name must be between 1 and 32 characters."
		case "Username":
			problems["username"] = "Username must be alphanumeric and between 8 and 32 characters."
		case "Password":
			problems["password"] = "Password must be at least 8 characters long and contain numbers/symbols."
		}
	}

	return problems
}