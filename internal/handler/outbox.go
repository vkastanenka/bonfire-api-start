package handler

import (
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/outbox"
	"net/http"

	"github.com/google/uuid"
)

type OutboxEventHandler struct {
	repo outbox.Repository
	bind *httpio.Bind
}

func NewOutboxEventHandler(repo outbox.Repository, bind *httpio.Bind) *OutboxEventHandler {
	return &OutboxEventHandler{repo: repo, bind: bind}
}

type OutboxEventGetByIDPath struct {
	ID uuid.UUID `path:"id" validate:"required,uuid"`
}

func (h *OutboxEventHandler) OutboxEventGetByID(w http.ResponseWriter, r *http.Request) error {
	var path OutboxEventGetByIDPath
	err := h.bind.Path(r, &path)
	if err != nil {
		return err
	}

	event, err := h.repo.Get(r.Context(), path.ID)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, ToOutboxEventResponse(event))
	return nil
}

type ResetAttemptsPath struct {
	ID uuid.UUID `path:"id" validate:"required,uuid"`
}

func (h *OutboxEventHandler) OutboxEventResetAttempts(w http.ResponseWriter, r *http.Request) error {
	var path ResetAttemptsPath
	err := h.bind.Path(r, &path)
	if err != nil {
		return err
	}

	event, err := h.repo.ResetAttempts(r.Context(), path.ID)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, ToOutboxEventResponse(event))
	return nil
}

type OutboxEventDeleteByIDPath struct {
	ID uuid.UUID `path:"id" validate:"required,uuid"`
}

func (h *OutboxEventHandler) Delete(w http.ResponseWriter, r *http.Request) error {
	var path OutboxEventDeleteByIDPath
	err := h.bind.Path(r, &path)
	if err != nil {
		return err
	}

	if err = h.repo.Delete(r.Context(), path.ID); err != nil {
		return err
	}

	httpio.RespondNoContent(w)
	return nil
}

func (h *OutboxEventHandler) OutboxEventPurgeProcessed(w http.ResponseWriter, r *http.Request) error {
	if err := h.repo.PurgeProcessed(r.Context()); err != nil {
		return err
	}

	httpio.RespondNoContent(w)
	return nil
}
