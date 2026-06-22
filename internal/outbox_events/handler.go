package outbox_events

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/repository"
	"bonfire-api/internal/validator"
	"net/http"
	"strconv"

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

// Count
func (h *Handler) Count(w http.ResponseWriter, r *http.Request) error {
	count, err := h.service.Count(r.Context())
	if err != nil {
		return apperr.NewDBError(err)
	}

	httpio.RespondOK(w, CountRes{Count: count}, "Count retrieved.")
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
	// Parse Query Parameters
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	// Default values
	var limit int32 = 20
	var offset int32 = 0

	// Safely parse strings to int32
	if l, err := strconv.ParseInt(limitStr, 10, 32); err == nil && l > 0 {
		limit = int32(l)
	}
	if o, err := strconv.ParseInt(offsetStr, 10, 32); err == nil && o >= 0 {
		offset = int32(o)
	}

	params := repository.OutboxEventListParams{
		Limit:  limit,
		Offset: offset,
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

// GetByID
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) error {
	// 1. Extract ID from URL
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return apperr.New(apperr.CodeBadRequest, "Invalid ID format")
	}

	// 2. Call service directly, no JSON binding needed!
	row, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, row, "Get by id ok.")
	return nil
}

// UpdateReq (Removed ID field)
type UpdateReq struct {
	EventType   string `json:"event_type" validate:"required"`
	Payload     []byte `json:"payload" validate:"required"`
	MaxAttempts int32  `json:"max_attempts"`
}

// UpdateByID
func (h *Handler) UpdateByID(w http.ResponseWriter, r *http.Request) error {
	// 1. Extract ID from URL
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return apperr.New(apperr.CodeBadRequest, "Invalid ID format")
	}

	// 2. Bind the rest of the payload from the JSON body
	req, err := httpio.BindJSON[UpdateReq](w, r, h.val)
	if err != nil {
		return err
	}

	params := repository.OutboxEventUpdateByIDParams{
		EventType:   req.EventType,
		Payload:     req.Payload,
		MaxAttempts: req.MaxAttempts,
	}

	// 3. Pass the parsed URL ID and the JSON params to the service
	row, err := h.service.UpdateByID(r.Context(), id, params)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, row, "Updated outbox event.")
	return nil
}

// DeleteByID
func (h *Handler) DeleteByID(w http.ResponseWriter, r *http.Request) error {
	// 1. Extract ID from URL
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return apperr.New(apperr.CodeBadRequest, "Invalid ID format")
	}

	// 2. Call service
	err = h.service.DeleteByID(r.Context(), id)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, struct{}{}, "Delete by id ok.")
	return nil
}
