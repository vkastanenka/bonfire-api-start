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
