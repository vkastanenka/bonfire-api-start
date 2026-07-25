package handler

import (
	"context"
	"net/http"

	"bonfire-api/internal/httpio"
	"bonfire-api/internal/presence"
	"bonfire-api/internal/user"

	"github.com/google/uuid"
)

type UserService interface {
	CheckAvailability(ctx context.Context, email user.Email, username user.Username) (emailAvail bool, usernameAvail bool, err error)
	Get(ctx context.Context, id uuid.UUID) (*user.User, error)
	GetByEmail(ctx context.Context, email user.Email) (*user.User, error)
	GetByUsername(ctx context.Context, username user.Username) (*user.User, error)
	SetPreferredPresence(ctx context.Context, id uuid.UUID, presence *presence.Presence) (*user.User, error)
	UpdateProfile(ctx context.Context, p user.UpdateProfileParams) (*user.User, error)
}

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

func (h *User) Get(w http.ResponseWriter, r *http.Request) error {
	var path UserGetPath
	err := h.bind.Path(r, &path)
	if err != nil {
		return err
	}

	userRow, err := h.service.Get(r.Context(), path.ID)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, ToUserResponse(*userRow))
	return nil
}
