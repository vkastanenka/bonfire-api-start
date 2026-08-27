package handler

import (
	"net/http"

	"bonfire-api/internal/httpio"

	"github.com/google/uuid"
)

type SessionHandler struct {
	service SessionService
	bind    *httpio.Bind
}

func NewSessionHandler(service SessionService, bind *httpio.Bind) *SessionHandler {
	return &SessionHandler{
		service: service,
		bind:    bind,
	}
}

type SessionPath struct {
	SessionID uuid.UUID `path:"sessionId" validate:"required,uuid"`
}

func (h *SessionHandler) ListValid(w http.ResponseWriter, r *http.Request) error {
	userID, err := httpio.CtxGetUserID(r.Context())
	if err != nil {
		return err
	}

	sessions, err := h.service.ListValidByUserID(r.Context(), userID.UUID())
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, sessions)
	return nil
}

func (h *SessionHandler) Revoke(w http.ResponseWriter, r *http.Request) error {
	userID, err := httpio.CtxGetUserID(r.Context())
	if err != nil {
		return err
	}

	var path SessionPath
	if err := h.bind.Path(r, &path); err != nil {
		return err
	}

	if err := h.service.Revoke(r.Context(), path.SessionID, userID.UUID()); err != nil {
		return err
	}

	httpio.RespondNoContent(w)
	return nil
}

func (h *SessionHandler) RevokeAll(w http.ResponseWriter, r *http.Request) error {
	userID, err := httpio.CtxGetUserID(r.Context())
	if err != nil {
		return err
	}

	if err := h.service.RevokeAll(r.Context(), userID.UUID()); err != nil {
		return err
	}

	httpio.RespondNoContent(w)
	return nil
}
