package outbox_events

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/repository"
	"bonfire-api/internal/validator"
	"net/http"

	"github.com/google/uuid"
)

type Handler struct {
	service *Service
	val     *validator.Validator
}

func NewHandler(service *Service, val *validator.Validator) *Handler {
	return &Handler{service: service, val: val}
}

// Ping
func (h *Handler) Ping(w http.ResponseWriter, r *http.Request) error {
	httpio.RespondJSON(w, http.StatusOK, map[string]string{
		"status": "healthy",
	})

	return nil
}

// Create
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) error {
	req, err := httpio.BindJSON[CreateReq](w, r, h.val)
	if err != nil {
		return err
	}

	params := repository.OutboxEventCreateParams{
		EventType: req.EventType,
		Payload:   req.Payload,
	}

	row, err := h.service.Create(r.Context(), params)
	if err != nil {
		return err
	}

	httpio.RespondCreated(w, row, "Created outbox event.")
	return nil
}

type ListReq struct {
	Limit  int32 `json:"limit"`
	Offset int32 `json:"offset"`
}

// List
func (h *Handler) List(w http.ResponseWriter, r *http.Request) error {
	req, err := httpio.BindJSON[ListReq](w, r, h.val)
	if err != nil {
		return err
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}

	params := repository.OutboxEventListParams{
		Limit:  limit,
		Offset: req.Offset,
	}

	events, err := h.service.List(r.Context(), params)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, events, "Events retrieved.")
	return nil
}

type CountRes struct {
	Count int64 `json:"count"`
}

// Count
func (h *Handler) Count(w http.ResponseWriter, r *http.Request) error {
	count, err := h.service.Count(r.Context())
	if err != nil {
		return apperr.NewDBError(err)
	}

	httpio.RespondOK(w, CountRes{Count: count}, "Count retrieved.")
	return nil
}

// GetByID
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) error {
	req, err := httpio.BindJSON[GetByIDReq](w, r, h.val)
	if err != nil {
		return err
	}

	row, err := h.service.GetByID(r.Context(), req.ID)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, row, "Get by id ok.")
	return nil
}

type UpdateReq struct {
	ID          uuid.UUID `json:"id" validate:"required"`
	EventType   string    `json:"event_type" validate:"required"`
	Payload     []byte    `json:"payload" validate:"required"`
	MaxAttempts int32     `json:"max_attempts"`
}

// UpdateByID
func (h *Handler) UpdateByID(w http.ResponseWriter, r *http.Request) error {
	req, err := httpio.BindJSON[UpdateReq](w, r, h.val)
	if err != nil {
		return err
	}

	params := repository.OutboxEventUpdateByIDParams{
		EventType:   req.EventType,
		Payload:     req.Payload,
		MaxAttempts: req.MaxAttempts,
	}

	row, err := h.service.UpdateByID(r.Context(), req.ID, params)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, row, "Updated outbox event.")
	return nil
}

// DeleteByID
func (h *Handler) DeleteByID(w http.ResponseWriter, r *http.Request) error {
	req, err := httpio.BindJSON[GetByIDReq](w, r, h.val)
	if err != nil {
		return err
	}

	err = h.service.DeleteByID(r.Context(), req.ID)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, struct{}{}, "Delete by id ok.")
	return nil
}
