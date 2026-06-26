package session

import (
	"net/http"
	"strings"

	"bonfire-api/internal/httpio"
	"bonfire-api/internal/sanitize"
	"bonfire-api/internal/validator"

	"github.com/google/uuid"
)

// --- Session Handler ---

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

// --- Session Handler Count GET ---

type CountRes struct {
	Count int64 `json:"count"`
}

func (h *Handler) Count(w http.ResponseWriter, r *http.Request) error {
	count, err := h.service.Count(r.Context())
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, CountRes{Count: count}, "")
	return nil
}

// ==========================================
// LIST
// ==========================================

// --- Session Handler List GET  ---

type ListReq struct {
	UserID uuid.UUID `form:"user_id" validate:"required"`
	Status string    `form:"status"  validate:"omitempty,oneof=active blocked expired"`
}

func (r *ListReq) Sanitize() {
	r.Status = strings.ToLower(sanitize.SanitizeText(r.Status))
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) error {
	// Get Query
	req, err := httpio.BindQuery[ListReq](r, h.validator)
	if err != nil {
		return err
	}

	// Call service
	views, err := h.service.List(r.Context(), ListParams{
		UserID: req.UserID,
		Status: req.Status,
	})
	if err != nil {
		return err
	}

	// Respond
	httpio.RespondOK(w, r, views, "")
	return nil
}

// ==========================================
// GET
// ==========================================

// --- Session Handler GetByID GET  ---

type GetByIDPath struct {
	ID uuid.UUID `path:"id"      validate:"required"`
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) error {
	path, err := httpio.BindPath[GetByIDPath](r, h.validator)
	if err != nil {
		return err
	}

	view, err := h.service.GetByID(r.Context(), path.ID)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, view, "")
	return nil
}

// ==========================================
// UPDATE
// ==========================================

// --- Session Handler UpdateLastSeen PATCH  ---

type UpdateLastSeenPath struct {
	ID uuid.UUID `path:"id"      validate:"required"`
}

func (h *Handler) UpdateLastSeen(w http.ResponseWriter, r *http.Request) error {
	path, err := httpio.BindPath[UpdateLastSeenPath](r, h.validator)
	if err != nil {
		return err
	}

	view, err := h.service.UpdateLastSeen(r.Context(), path.ID)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, view, "ok")
	return nil
}

// --- Session Handler MarkBlocked POST  ---

type MarkBlockedPath struct {
	ID uuid.UUID `path:"id"      validate:"required"`
}

func (h *Handler) MarkBlocked(w http.ResponseWriter, r *http.Request) error {
	path, err := httpio.BindPath[MarkBlockedPath](r, h.validator)
	if err != nil {
		return err
	}

	view, err := h.service.MarkBlocked(r.Context(), path.ID)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, view, "ok")
	return nil
}

// ==========================================
// DELETE
// ==========================================

// --- Session Handler Delete DELETE  ---

type DeletePath struct {
	ID uuid.UUID `path:"id"      validate:"required"`
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) error {
	path, err := httpio.BindPath[DeletePath](r, h.validator)
	if err != nil {
		return err
	}

	if err := h.service.Delete(r.Context(), path.ID); err != nil {
		return err
	}

	httpio.RespondNoContent(w)
	return nil
}

// --- Session Handler DeleteAllExcept DELETE  ---

type DeleteAllExceptQuery = struct {
	ID     uuid.UUID `form:"id"      validate:"required"`
	UserID uuid.UUID `form:"user_id"  validate:"required"`
}

func (h *Handler) DeleteAllExcept(w http.ResponseWriter, r *http.Request) error {
	// Get Query
	query, err := httpio.BindQuery[DeleteAllExceptQuery](r, h.validator)
	if err != nil {
		return err
	}

	if err := h.service.DeleteAllExcept(r.Context(), query.ID, query.UserID); err != nil {
		return err
	}

	httpio.RespondNoContent(w)
	return nil
}
