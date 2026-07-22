package handler

import (
	"bonfire-api/internal/httpio"
	"net/http"

	"github.com/google/uuid"
)

type UserHandler struct {
	service UserService
	bind    *httpio.Bind
}

func NewUserHandler(repo UserService, bind *httpio.Bind) *UserHandler {
	return &UserHandler{
		service: repo,
		bind:    bind,
	}
}

type UserGetPath struct {
	ID uuid.UUID `path:"id" validate:"required,uuid"`
}

func (h *UserHandler) UserGet(w http.ResponseWriter, r *http.Request) error {
	var path UserGetPath
	err := h.bind.JSON(w, r, &path)
	if err != nil {
		return err
	}

	userRow, err := h.service.Get(r.Context(), path.ID)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, ToUserResponse(userRow))
	return nil
}

func (h *UserHandler) UserGetMe(w http.ResponseWriter, r *http.Request) error {
	userID, err := httpio.CtxGetUserID(r.Context())
	if err != nil {
		return err
	}

	data, err := h.service.Get(r.Context(), userID)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, data)
	return nil
}
