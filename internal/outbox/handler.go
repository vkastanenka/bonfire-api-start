package outbox

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/httpio"
	"net/http"
	"strconv"

	"github.com/google/uuid"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// ==========================================
// META
// ==========================================

// Ping GET
func (h *Handler) Ping(w http.ResponseWriter, r *http.Request) error {
	httpio.RespondOK(w, r, PingRes{Status: "healthy"}, PingOK)

	return nil
}

// Count GET
func (h *Handler) Count(w http.ResponseWriter, r *http.Request) error {
	count, err := h.service.Count(r.Context())
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, CountRes{Count: count}, CountOK)
	return nil
}

// ==========================================
// LIST
// ==========================================

// List GET
func (h *Handler) List(w http.ResponseWriter, r *http.Request) error {
	limitStr := r.URL.Query().Get("limit")
	cursorStr := r.URL.Query().Get("cursor")

	var limit int32 = 20
	if l, err := strconv.ParseInt(limitStr, 10, 32); err == nil && l > 0 {
		limit = int32(l)
	}

	var cursor *uuid.UUID
	if cursorStr != "" {
		parsed, err := uuid.Parse(cursorStr)
		if err != nil {
			return apperr.New(apperr.CodeBadRequest, ErrInvalidCursor)
		}
		cursor = &parsed
	}

	events, err := h.service.List(r.Context(), ListParams{
		Limit:  limit,
		Cursor: cursor,
	})
	if err != nil {
		return err
	}

	var nextCursor *string
	if len(events) == int(limit) {
		lastEvent := events[len(events)-1]
		strCursor := lastEvent.ID.String()
		nextCursor = &strCursor
	}

	httpio.RespondCursorList(w, r, events, ListOK, httpio.CursorPagination{
		NextCursor: nextCursor,
		PageSize:   int32(len(events)),
	})
	return nil
}

// ==========================================
// GET
// ==========================================

// GetByID GET
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) error {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return apperr.New(apperr.CodeBadRequest, ErrInvalidID)
	}

	row, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, row, GetByIDOK)
	return nil
}

// ==========================================
// UPDATE
// ==========================================

// ResetAttempts POST
func (h *Handler) ResetAttempts(w http.ResponseWriter, r *http.Request) error {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return apperr.New(apperr.CodeBadRequest, ErrInvalidID)
	}

	row, err := h.service.ResetAttempts(r.Context(), id)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, row, ResetAttemptsOK)
	return nil
}

// ==========================================
// DELETE
// ==========================================

// DeleteByID DELETE
func (h *Handler) DeleteByID(w http.ResponseWriter, r *http.Request) error {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return apperr.New(apperr.CodeBadRequest, ErrInvalidID)
	}

	if err := h.service.DeleteByID(r.Context(), id); err != nil {
		return err
	}

	httpio.RespondNoContent(w)
	return nil
}

// PurgeProcessed POST
func (h *Handler) PurgeProcessed(w http.ResponseWriter, r *http.Request) error {
	if err := h.service.PurgeProcessed(r.Context()); err != nil {
		return err
	}
	httpio.RespondNoContent(w)
	return nil
}
