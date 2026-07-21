package handler

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/user"
	"net/http"

	"github.com/google/uuid"
)

type UserHandler struct {
	repo user.Repository
}

func NewUserHandler(repo user.Repository) *UserHandler {
	return &UserHandler{
		repo: repo,
	}
}

type GetByIDPath struct {
	ID uuid.UUID `path:"id" validate:"required,idSchema"`
}

func (h *UserHandler) UserGetByID(w http.ResponseWriter, r *http.Request) error {
	path, err := httpio.BindPath[GetByIDPath](nil, r)
	if err != nil {
		return err
	}

	userRow, err := h.repo.Get(r.Context(), path.ID)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, ToUserResponse(userRow))
	return nil
}

type UserGetQuery struct {
	Email    *string `form:"email" validate:"omitempty,emailSchema"`
	Username *string `form:"username"  validate:"omitempty,userUsernameSchema"`
}

func (h *UserHandler) UserGet(w http.ResponseWriter, r *http.Request) error {
	query, err := httpio.BindQuery[UserGetQuery](nil, r)
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
		userRow, err = h.repo.GetByEmail(r.Context(), email)
	} else if username != "" {
		userRow, err = h.repo.GetByUsername(r.Context(), username)
	} else {
		return apperr.NewInvalidArgument(nil, apperr.WithMsg("either email or username query parameter must be provided"))
	}

	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, ToUserResponse(userRow))
	return nil
}

func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) error {
	userID, err := httpio.GetCtxUserID(r.Context())
	if err != nil {
		return err
	}

	data, err := h.repo.Get(r.Context(), userID)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, data)
	return nil
}

// type ListRelationshipsQuery struct {
// 	Type *string `form:"type" validate:"omitempty,oneof=friends pending blocked"`
// }

// func (h *Handler) ListRelationships(w http.ResponseWriter, r *http.Request) error {
// 	userID, err := httpio.GetCtxUserID(r.Context())
// 	if err != nil {
// 		return err
// 	}

// 	query, err := httpio.BindQuery[ListRelationshipsQuery](r)
// 	if err != nil {
// 		return err
// 	}

// 	filterType := relationship.TypeUnknown
// 	if query.Type != nil {
// 		switch *query.Type {
// 		case "friends":
// 			filterType = relationship.TypeFriends
// 		case "pending":
// 			filterType = relationship.TypePending
// 		case "blocked":
// 			filterType = relationship.TypeBlocked
// 		}
// 	}

// 	data, err := h.relationship.List(r.Context(), userID, filterType)
// 	if err != nil {
// 		return err
// 	}

// 	httpio.RespondOK(w, r, data)
// 	return nil
// }

// type RelationshipPath struct {
// 	ID uuid.UUID `path:"id" validate:"required"`
// }

// func (h *Handler) UpsertRelationship(w http.ResponseWriter, r *http.Request) error {
// 	userID, err := httpio.GetCtxUserID(r.Context())
// 	if err != nil {
// 		return err
// 	}

// 	path, err := httpio.BindPath[RelationshipPath](r)
// 	if err != nil {
// 		return err
// 	}

// 	err = h.relationship.SendFriendRequest(r.Context(), userID, path.ID)
// 	if err != nil {
// 		return err
// 	}

// 	httpio.RespondNoContent(w)
// 	return nil
// }

// func (h *Handler) BlockUser(w http.ResponseWriter, r *http.Request) error {
// 	userID, err := httpio.GetCtxUserID(r.Context())
// 	if err != nil {
// 		return err
// 	}

// 	path, err := httpio.BindPath[RelationshipPath](r)
// 	if err != nil {
// 		return err
// 	}

// 	err = h.relationship.Block(r.Context(), userID, path.ID)
// 	if err != nil {
// 		return err
// 	}

// 	httpio.RespondNoContent(w)
// 	return nil
// }

// func (h *Handler) DeleteRelationship(w http.ResponseWriter, r *http.Request) error {
// 	userID, err := httpio.GetCtxUserID(r.Context())
// 	if err != nil {
// 		return err
// 	}

// 	path, err := httpio.BindPath[RelationshipPath](r)
// 	if err != nil {
// 		return err
// 	}

// 	err = h.relationship.DeleteVerified(r.Context(), userID, path.ID)
// 	if err != nil {
// 		return err
// 	}

// 	httpio.RespondNoContent(w)
// 	return nil
// }
