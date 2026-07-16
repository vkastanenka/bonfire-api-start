package me

import (
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/relationship"
	"net/http"

	"github.com/google/uuid"
)

type Handler struct {
	service      *Service
	relationship *relationship.Service
}

func NewHandler(service *Service, relationship *relationship.Service) *Handler {
	return &Handler{service: service, relationship: relationship}
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) error {
	userID, err := httpio.GetCtxUserID(r.Context())
	if err != nil {
		return err
	}

	data, err := h.service.GetByID(r.Context(), userID)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, data)
	return nil
}

type ListRelationshipsQuery struct {
	Type *string `form:"type" validate:"omitempty,oneof=friends pending blocked"`
}

func (h *Handler) ListRelationships(w http.ResponseWriter, r *http.Request) error {
	userID, err := httpio.GetCtxUserID(r.Context())
	if err != nil {
		return err
	}

	query, err := httpio.BindQuery[ListRelationshipsQuery](r)
	if err != nil {
		return err
	}

	filterType := relationship.TypeUnknown
	if query.Type != nil {
		switch *query.Type {
		case "friends":
			filterType = relationship.TypeFriends
		case "pending":
			filterType = relationship.TypePending
		case "blocked":
			filterType = relationship.TypeBlocked
		}
	}

	data, err := h.relationship.List(r.Context(), userID, filterType)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, data)
	return nil
}

type RelationshipPath struct {
	ID uuid.UUID `path:"id" validate:"required"`
}

func (h *Handler) UpsertRelationship(w http.ResponseWriter, r *http.Request) error {
	userID, err := httpio.GetCtxUserID(r.Context())
	if err != nil {
		return err
	}

	path, err := httpio.BindPath[RelationshipPath](r)
	if err != nil {
		return err
	}

	err = h.relationship.SendFriendRequest(r.Context(), userID, path.ID)
	if err != nil {
		return err
	}

	httpio.RespondNoContent(w)
	return nil
}

func (h *Handler) BlockUser(w http.ResponseWriter, r *http.Request) error {
	userID, err := httpio.GetCtxUserID(r.Context())
	if err != nil {
		return err
	}

	path, err := httpio.BindPath[RelationshipPath](r)
	if err != nil {
		return err
	}

	err = h.relationship.Block(r.Context(), userID, path.ID)
	if err != nil {
		return err
	}

	httpio.RespondNoContent(w)
	return nil
}

func (h *Handler) DeleteRelationship(w http.ResponseWriter, r *http.Request) error {
	userID, err := httpio.GetCtxUserID(r.Context())
	if err != nil {
		return err
	}

	path, err := httpio.BindPath[RelationshipPath](r)
	if err != nil {
		return err
	}

	err = h.relationship.DeleteVerified(r.Context(), userID, path.ID)
	if err != nil {
		return err
	}

	httpio.RespondNoContent(w)
	return nil
}
