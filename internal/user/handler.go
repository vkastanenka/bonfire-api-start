package user

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/validator"
	"net/http"
	"strconv"

	"github.com/google/uuid"
)

type Handler struct {
	service   *Service
	validator *validator.Validator
}

func NewHandler(service *Service, validator *validator.Validator) *Handler {
	return &Handler{service: service, validator: validator}
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

// CheckAvailability GET
func (h *Handler) CheckAvailability(w http.ResponseWriter, r *http.Request) error {
	params := CheckAvailabilityParams{
		Email:    r.URL.Query().Get("email"),
		Username: r.URL.Query().Get("username"),
	}

	res, err := h.service.CheckAvailability(r.Context(), params)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, res, CheckAvailabilityOK)
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

	rows, err := h.service.List(r.Context(), ListParams{
		Limit:  limit,
		Cursor: cursor,
	})
	if err != nil {
		return err
	}

	var nextCursor *string
	if len(rows) == int(limit) {
		lastEvent := rows[len(rows)-1]
		strCursor := lastEvent.ID.String()
		nextCursor = &strCursor
	}

	httpio.RespondCursorList(w, r, rows, ListOK, httpio.CursorPagination{
		NextCursor: nextCursor,
		PageSize:   int32(len(rows)),
	})
	return nil
}

// ListUnverified GET
func (h *Handler) ListUnverified(w http.ResponseWriter, r *http.Request) error {
	limitStr := r.URL.Query().Get("limit")
	var limit int32 = 20
	if l, err := strconv.ParseInt(limitStr, 10, 32); err == nil && l > 0 {
		limit = int32(l)
	}

	rows, err := h.service.ListUnverified(r.Context(), limit)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, rows, ListUnverifiedOK)
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

// GetByEmail GET
func (h *Handler) GetByEmail(w http.ResponseWriter, r *http.Request) error {
	email := r.PathValue("email")
	if email == "" {
		return apperr.New(apperr.CodeBadRequest, "email required")
	}

	row, err := h.service.GetByEmail(r.Context(), email)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, row, GetByEmailOK)
	return nil
}

// GetByUsername GET
func (h *Handler) GetByUsername(w http.ResponseWriter, r *http.Request) error {
	username := r.PathValue("username")
	if username == "" {
		return apperr.New(apperr.CodeBadRequest, "username required")
	}

	row, err := h.service.GetByUsername(r.Context(), username)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, row, GetByUsernameOK)
	return nil
}

// ==========================================
// DELETE
// ==========================================

// DeleteByID DELETE
func (h *Handler) DeleteByID(w http.ResponseWriter, r *http.Request) error {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		return apperr.New(apperr.CodeBadRequest, "invalid id")
	}

	if err := h.service.DeleteByID(r.Context(), id); err != nil {
		return err
	}

	httpio.RespondNoContent(w)
	return nil
}

// DeleteByEmail DELETE
func (h *Handler) DeleteByEmail(w http.ResponseWriter, r *http.Request) error {
	email := r.PathValue("email")
	if email == "" {
		return apperr.New(apperr.CodeBadRequest, "email required")
	}

	if err := h.service.DeleteByEmail(r.Context(), email); err != nil {
		return err
	}

	httpio.RespondNoContent(w)
	return nil
}
