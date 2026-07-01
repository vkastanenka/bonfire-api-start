package guild

import (
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/validator"
	"net/http"

	"github.com/google/uuid"
)

type Handler struct {
	service   *Service
	validator *validator.Validator
}

func NewHandler(service *Service, validator *validator.Validator) *Handler {
	return &Handler{service: service, validator: validator}
}

type GetGuildPath struct {
	ID uuid.UUID `path:"id" validate:"required"`
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) error {
	path, err := httpio.BindPath[GetGuildPath](r, h.validator)
	if err != nil {
		return err
	}

	view, err := h.service.GetFull(r.Context(), path.ID)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, view, "")
	return nil
}

type GetPermissionsPath struct {
	GuildID uuid.UUID `path:"guild_id" validate:"required"`
	UserID  uuid.UUID `path:"user_id"  validate:"required"`
}

func (h *Handler) GetPermissions(w http.ResponseWriter, r *http.Request) error {
	path, err := httpio.BindPath[GetPermissionsPath](r, h.validator)
	if err != nil {
		return err
	}

	perms, err := h.service.GetEffectivePermissions(r.Context(), path.GuildID, path.UserID)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, map[string]int64{"permissions": perms}, "")
	return nil
}
