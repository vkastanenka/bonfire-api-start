package me

import (
	"bonfire-api/internal/httpio"
	"net/http"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) error {
	userID, err := httpio.GetCtxUserID(r.Context())
	if err != nil {
		return err
	}

	data, err := h.service.GetByID(r.Context(), userID)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, data)
	return nil
}
