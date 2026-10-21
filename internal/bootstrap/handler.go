package bootstrap

import (
	"net/http"

	"bonfire-api/internal/appctx"
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/user"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	service *Service
	bind    *httpio.Bind
}

func NewHandler(service *Service, bind *httpio.Bind) *Handler {
	return &Handler{
		service: service,
		bind:    bind,
	}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/users/@me", httpio.ToHTTPErr(h.bootstrap))
}

type bootstrapResponse struct {
	Me *user.MeDTO `json:"me"`
}

func (h *Handler) bootstrap(w http.ResponseWriter, r *http.Request) error {
	claims, err := appctx.GetClaims(r.Context())
	if err != nil {
		return err
	}

	result, err := h.service.Bootstrap(r.Context(), Params{
		UserID: claims.UserID,
	})
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, bootstrapResponse{
		Me: user.NewMeDTO(result.User),
	})
	return nil
}
