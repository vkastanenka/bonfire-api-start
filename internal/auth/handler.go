package auth

import (
	"bonfire-api/internal/validator"
)

// --- HANDLER TYPES ---

type Handler struct {
	service   *Service
	validator *validator.Validator
}

// --- HANDLER INITIALIZATION ---

func NewHandler(service *Service, validator *validator.Validator) *Handler {
	return &Handler{service: service, validator: validator}
}
