package handler

import (
	"bonfire-api/internal/httpio"
	"net/http"

	"github.com/google/uuid"
)

type User struct {
	service UserService
	bind    *httpio.Bind
}

func NewUser(repo UserService, bind *httpio.Bind) *User {
	return &User{
		service: repo,
		bind:    bind,
	}
}

type UserGetPath struct {
	ID uuid.UUID `path:"id" validate:"required,uuid"`
}

func (h *User) UserGet(w http.ResponseWriter, r *http.Request) error {
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

func (h *User) UserGetMe(w http.ResponseWriter, r *http.Request) error {
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
