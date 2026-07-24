package handler

import (
	"bonfire-api/internal/errs"
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

	outboxID, err := outbox.NewEventID(path.ID)
	if err != nil {
		return errs.InvalidArgument("Invalid ID.").
			Wrap(err)
	}

	event, err := h.repo.Get(r.Context(), outboxID)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, ToOutboxEventResponse(*event))
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

	outboxID, err := outbox.NewEventID(path.ID)
	if err != nil {
		return errs.InvalidArgument("Invalid ID.").
			Wrap(err)
	}

	if err = h.repo.Delete(r.Context(), outboxID); err != nil {
		return err
	}

	httpio.RespondNoContent(w)
	return nil
}
