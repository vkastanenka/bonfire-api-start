package outbox_events

import (
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/repository"
	"bonfire-api/internal/validator"
	"net/http"
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

// List retrieves a batch of events
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) error {
	req, err := httpio.BindJSON[repository.OutboxEventCreateParams](w, r, h.val)
	if err != nil {
		return err
	}

	row, err := h.service.Create(r.Context(), req)
	if err != nil {
		return err
	}

	httpio.RespondCreated(w, row, "Created outbox event.")
	return nil
}

func (h *Handler) CreateBatch(w http.ResponseWriter, r *http.Request) error {
	// 1. Bind slice of events
	req, err := httpio.BindJSON[[]repository.OutboxEventCreateParams](w, r, h.val)
	if err != nil {
		return err
	}

	// 2. Call the service
	rows, err := h.service.CreateBatch(r.Context(), req)
	if err != nil {
		return err
	}

	// 3. Respond with the created slice
	httpio.RespondOK(w, rows, "Created outbox events.")
	return nil
}

// List retrieves a batch of events
func (h *Handler) List(w http.ResponseWriter, r *http.Request) error {
	req, err := httpio.BindJSON[ListReq](w, r, h.val)
	if err != nil {
		return err
	}

	views, err := h.service.List(r.Context(), ListParams{Limit: req.Limit, Offset: req.Offset})
	if err != nil {
		return err
	}

	httpio.RespondOK(w, views, "Events retrieved.")
	return nil
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) error {
	// 1. Bind and Validate
	req, err := httpio.BindJSON[GetByIDReq](w, r, h.val)
	if err != nil {
		return err
	}

	// 2. Call Service
	view, err := h.service.GetByID(r.Context(), GetByIDParams{ID: req.ID})
	if err != nil {
		return err
	}

	// 3. Respond with the DTO
	httpio.RespondOK(w, view, "Get by id ok.")
	return nil
}

// MarkProcessed updates the event status
func (h *Handler) MarkProcessed(w http.ResponseWriter, r *http.Request) error {
	req, err := httpio.BindJSON[IDReq](w, r, h.val)
	if err != nil {
		return err
	}

	if err := h.service.MarkProcessed(r.Context(), req.ID); err != nil {
		return err
	}

	httpio.RespondOK(w, nil, "Event marked as processed.")
	return nil
}

// RecordFailure updates status after a failed attempt
func (h *Handler) RecordFailure(w http.ResponseWriter, r *http.Request) error {
	req, err := httpio.BindJSON[RecordFailureReq](w, r, h.val)
	if err != nil {
		return err
	}

	err = h.service.RecordFailure(r.Context(), RecordFailureParams{
		ID:    req.ID,
		Error: req.Error,
	})
	if err != nil {
		return err
	}

	httpio.RespondOK(w, nil, "Failure recorded.")
	return nil
}

// MarkDeadLetter manually moves to dead letter queue
func (h *Handler) MarkDeadLetter(w http.ResponseWriter, r *http.Request) error {
	req, err := httpio.BindJSON[RecordFailureReq](w, r, h.val)
	if err != nil {
		return err
	}

	if err := h.service.MarkDeadLetter(r.Context(), req.ID, req.Error); err != nil {
		return err
	}

	httpio.RespondOK(w, nil, "Event moved to dead letter.")
	return nil
}

// CountPending provides current queue health
func (h *Handler) CountPending(w http.ResponseWriter, r *http.Request) error {
	count, err := h.service.CountPending(r.Context())
	if err != nil {
		return err
	}

	httpio.RespondOK(w, map[string]int64{"count": count}, "Count retrieved.")
	return nil
}
