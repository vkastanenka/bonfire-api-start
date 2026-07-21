package http

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/user"
	"net/http"

	"github.com/google/uuid"
)

type UserHandler struct {
	repository user.Repository
	binder     *RequestBinder
}

func NewUserHandler(repository user.Repository, binder *RequestBinder) *UserHandler {
	return &UserHandler{
		repository: repository,
		binder:     binder,
	}
}

type GetByIDPath struct {
	ID uuid.UUID `path:"id" validate:"required,idSchema"`
}

func (h *UserHandler) GetByID(w http.ResponseWriter, r *http.Request) error {
	path, err := BindPath[GetByIDPath](h.binder, r)
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
	Email    *string `form:"email" validate:"omitempty,emailSchema"`
	Username *string `form:"username"  validate:"omitempty,userUsernameSchema"`
}

func (h *UserHandler) Get(w http.ResponseWriter, r *http.Request) error {
	query, err := BindQuery[GetQuery](h.binder, r)
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
		return apperr.NewInvalidArgument(nil, apperr.WithMsg("either email or username query parameter must be provided"))
	}

	if err != nil {
		return err
	}

	RespondOK(w, r, user.ToPublicView(userRow))
	return nil
}
