package auth

import (
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/validator"
	"net/http"
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

// --- PING CONSTANTS ---

const (
	MsgPingSuccess = "pong"
)

// --- PING DTO ---

type PingRes struct {
	Status string `json:"status"`
}

// --- PING HANDLER ---

// Ping
func (h *Handler) Ping(w http.ResponseWriter, r *http.Request) error {
	httpio.RespondOK(w, r, PingRes{Status: "healthy"}, MsgPingSuccess)
	return nil
}
