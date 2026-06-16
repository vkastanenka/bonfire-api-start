package auth

import (
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/validator"
	"net/http"
)

type AuthHandler struct {
	service *AuthService
	val     *validator.Validator
}

func NewHandler(service *AuthService, val *validator.Validator) *AuthHandler {
	return &AuthHandler{service: service, val: val}
}

// Ping confirms the auth routes are available
func (h *AuthHandler) Ping(w http.ResponseWriter, r *http.Request) error {
	httpio.RespondJSON(w, http.StatusOK, map[string]string{
		"status": "healthy",
	})

	return nil
}
