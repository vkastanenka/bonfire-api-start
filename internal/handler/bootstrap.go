package handler

import (
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/user"
	"net/http"
)

type BootstrapHandler struct {
	service BootstrapService
	bind    *httpio.Bind
}

func NewBootstrapHandler(service BootstrapService, bind *httpio.Bind) *BootstrapHandler {
	return &BootstrapHandler{service: service, bind: bind}
}

type BootstrapResponse struct {
	Me user.Me `json:"me"`
}

func (h *BootstrapHandler) Bootstrap(w http.ResponseWriter, r *http.Request) error {
	result, err := h.service.Bootstrap(r.Context())
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, BootstrapResponse{
		Me: user.ParseMe(result.User),
	})
	return nil
}
