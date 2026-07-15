package user

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/httpio"
	"net/http"

	"github.com/google/uuid"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

type GetByIDPath struct {
	ID uuid.UUID `path:"id" validate:"required"`
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) error {
	path, err := httpio.BindPath[GetByIDPath](r)
	if err != nil {
		return err
	}

	userRow, err := h.service.GetByID(r.Context(), path.ID)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, ToView(userRow))
	return nil
}

type GetQuery struct {
	Email    *string `form:"email" validate:"omitempty,email"`
	Username *string `form:"username"  validate:"omitempty"`
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) error {
	query, err := httpio.BindQuery[GetQuery](r)
	if err != nil {
		return err
	}

	var email, username string
	if query.Email != nil {
		email = *query.Email
	}
	if query.Username != nil {
		username = *query.Username
	}

	var userRow User

	if email != "" {
		userRow, err = h.service.GetByEmail(r.Context(), email)
	} else if username != "" {
		userRow, err = h.service.GetByUsername(r.Context(), username)
	} else {
		return apperr.NewInvalidInput(nil, "either email or username query parameter must be provided")
	}

	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, ToView(userRow))
	return nil
}
