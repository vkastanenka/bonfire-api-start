package user

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/repository"
	"net/http"

	"github.com/google/uuid"
)

// ==========================================
// RELATIONSHIPS
// ==========================================

type SendFriendRequestPath struct {
	ID uuid.UUID `path:"id"      validate:"required"`
}

// SendFriendRequest POST /users/{id}/friends
func (h *Handler) SendFriendRequest(w http.ResponseWriter, r *http.Request) error {
	userID, err := httpio.GetCtxUserID(r.Context())
	if err != nil {
		return apperr.New(apperr.CodeUnauthorized, apperr.CodeUnauthorized.Title(), apperr.WithErr(err))
	}

	path, err := httpio.BindPath[SendFriendRequestPath](r, h.validator)
	if err != nil {
		return err
	}

	if err := h.service.SendFriendRequest(r.Context(), userID, path.ID); err != nil {
		return err
	}

	httpio.RespondNoContent(w) // Why?
	return nil
}

type AcceptFriendRequestPath struct {
	ID uuid.UUID `path:"id"      validate:"required"`
}

// AcceptFriendRequest PUT /users/{id}/friends
func (h *Handler) AcceptFriendRequest(w http.ResponseWriter, r *http.Request) error {
	userID, err := httpio.GetCtxUserID(r.Context())
	if err != nil {
		return apperr.New(apperr.CodeUnauthorized, apperr.CodeUnauthorized.Title(), apperr.WithErr(err))
	}

	path, err := httpio.BindPath[AcceptFriendRequestPath](r, h.validator)
	if err != nil {
		return err
	}

	if err := h.service.AcceptFriendRequest(r.Context(), userID, path.ID); err != nil {
		return err
	}

	httpio.RespondNoContent(w)
	return nil
}

type BlockUserPath struct {
	ID uuid.UUID `path:"id"      validate:"required"`
}

// BlockUser POST /users/{id}/block
func (h *Handler) BlockUser(w http.ResponseWriter, r *http.Request) error {
	userID, err := httpio.GetCtxUserID(r.Context())
	if err != nil {
		return apperr.New(apperr.CodeUnauthorized, apperr.CodeUnauthorized.Title(), apperr.WithErr(err))
	}

	path, err := httpio.BindPath[BlockUserPath](r, h.validator)
	if err != nil {
		return err
	}

	if err := h.service.BlockUser(r.Context(), userID, path.ID); err != nil {
		return err
	}

	httpio.RespondNoContent(w)
	return nil
}

type RemoveRelationshipPath struct {
	ID uuid.UUID `path:"id"      validate:"required"`
}

// RemoveRelationship DELETE /users/{id}/friends
// (Also used to unblock, or cancel a pending request)
func (h *Handler) RemoveRelationship(w http.ResponseWriter, r *http.Request) error {
	userID, err := httpio.GetCtxUserID(r.Context())
	if err != nil {
		return apperr.New(apperr.CodeUnauthorized, apperr.CodeUnauthorized.Title(), apperr.WithErr(err))
	}

	path, err := httpio.BindPath[RemoveRelationshipPath](r, h.validator)
	if err != nil {
		return err
	}

	if err := h.service.RemoveRelationship(r.Context(), userID, path.ID); err != nil {
		return err
	}

	httpio.RespondNoContent(w)
	return nil
}

// ListFriends GET /friends
func (h *Handler) ListFriends(w http.ResponseWriter, r *http.Request) error {
	userID, err := httpio.GetCtxUserID(r.Context())
	if err != nil {
		return apperr.New(apperr.CodeUnauthorized, apperr.CodeUnauthorized.Title(), apperr.WithErr(err))
	}

	// Example for listing active friends.
	// You can read a query param like ?status=pending to route appropriately.
	friends, err := h.service.ListRelationships(r.Context(), userID, repository.RelationshipStatusFriends)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, friends, "friends_listed_ok")
	return nil
}
