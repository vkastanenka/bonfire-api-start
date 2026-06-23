package auth

import (
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/validator"
	"net/http"
)

type Handler struct {
	service   *Service
	validator *validator.Validator
}

func NewHandler(service *Service, validator *validator.Validator) *Handler {
	return &Handler{service: service, validator: validator}
}

// Ping
func (h *Handler) Ping(w http.ResponseWriter, r *http.Request) error {
	httpio.RespondOK(w, r, PingRes{Status: "healthy"}, PingOK)
	return nil
}
