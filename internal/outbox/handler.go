package outbox

import (
	"net/http"

	"bonfire-api/internal/httpio"

	"github.com/google/uuid"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

type GetByIDPath struct {
	ID uuid.UUID `path:"id" validate:"required"`
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) error {
	path, err := httpio.BindPath[GetByIDPath](r)
	if err != nil {
		return err
	}

	event, err := h.service.GetByID(r.Context(), path.ID)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, ToAuthView(event))
	return nil
}

type ResetAttemptsPath struct {
	ID uuid.UUID `path:"id" validate:"required"`
}

func (h *Handler) ResetAttempts(w http.ResponseWriter, r *http.Request) error {
	path, err := httpio.BindPath[ResetAttemptsPath](r)
	if err != nil {
		return err
	}

	event, err := h.service.ResetAttempts(r.Context(), path.ID)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, ToAuthView(event))
	return nil
}

type DeleteByIDPath struct {
	ID uuid.UUID `path:"id" validate:"required"`
}

func (h *Handler) DeleteByID(w http.ResponseWriter, r *http.Request) error {
	path, err := httpio.BindPath[DeleteByIDPath](r)
	if err != nil {
		return err
	}

	if err = h.service.DeleteByID(r.Context(), path.ID); err != nil {
		return err
	}

	httpio.RespondNoContent(w)
	return nil
}

func (h *Handler) PurgeProcessed(w http.ResponseWriter, r *http.Request) error {
	if err := h.service.PurgeProcessed(r.Context()); err != nil {
		return err
	}

	httpio.RespondNoContent(w)
	return nil
}
