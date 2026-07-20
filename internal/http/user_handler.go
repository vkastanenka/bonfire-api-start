package http

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/user"
	"net/http"

	"github.com/google/uuid"
)

type UserHandler struct {
	repository user.Repository
}

func NewUserHandler(repository user.Repository) *UserHandler {
	return &UserHandler{repository: repository}
}

type GetByIDPath struct {
	ID uuid.UUID `path:"id" validate:"required"`
}

func (h *UserHandler) GetByID(w http.ResponseWriter, r *http.Request) error {
	path, err := BindPath[GetByIDPath](r)
	if err != nil {
		return err
	}

	userRow, err := h.repository.Get(r.Context(), path.ID)
	if err != nil {
		return err
	}

	RespondOK(w, r, user.ToPublicView(userRow))
	return nil
}

type GetQuery struct {
	Email    *string `form:"email" validate:"omitempty,email"`
	Username *string `form:"username"  validate:"omitempty"`
}

func (h *UserHandler) Get(w http.ResponseWriter, r *http.Request) error {
	query, err := BindQuery[GetQuery](r)
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

	var userRow user.User

	if email != "" {
		userRow, err = h.repository.GetByEmail(r.Context(), email)
	} else if username != "" {
		userRow, err = h.repository.GetByUsername(r.Context(), username)
	} else {
		return apperr.NewInvalidArgument(nil, apperr.WithMessage("either email or username query parameter must be provided"))
	}

	if err != nil {
		return err
	}

	RespondOK(w, r, user.ToPublicView(userRow))
	return nil
}
