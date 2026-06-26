package session

import (
	"net/http"

	"bonfire-api/internal/apperr"
	"bonfire-api/internal/httpio"
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

// --- Session Handler ListActiveByUserID GET  ---

func (h *Handler) ListActiveByUserID(w http.ResponseWriter, r *http.Request) error {
	userIDStr := r.PathValue("userId")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return apperr.New(apperr.CodeBadRequest, "invalid user id")
	}

	views, err := h.service.ListActiveByUserID(r.Context(), userID)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, views, "")
	return nil
}

// ==========================================
// GET
// ==========================================

// --- Session Handler GetByID GET  ---

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) error {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return apperr.New(apperr.CodeBadRequest, "invalid session id")
	}

	view, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, view, "")
	return nil
}

// --- Session Handler GetByRefreshToken GET  ---

func (h *Handler) GetByRefreshToken(w http.ResponseWriter, r *http.Request) error {
	token := r.URL.Query().Get("token")
	if token == "" {
		return apperr.New(apperr.CodeBadRequest, "refresh token query parameter required")
	}

	view, err := h.service.GetByRefreshToken(r.Context(), token)
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

func (h *Handler) UpdateLastSeen(w http.ResponseWriter, r *http.Request) error {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return apperr.New(apperr.CodeBadRequest, "invalid session id")
	}

	view, err := h.service.UpdateLastSeen(r.Context(), id)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, view, "ok")
	return nil
}

// --- Session Handler MarkBlocked POST  ---

func (h *Handler) MarkBlocked(w http.ResponseWriter, r *http.Request) error {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return apperr.New(apperr.CodeBadRequest, "invalid session id")
	}

	view, err := h.service.MarkBlocked(r.Context(), id)
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

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) error {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return apperr.New(apperr.CodeBadRequest, "invalid session id")
	}

	userIDStr := r.PathValue("userId")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return apperr.New(apperr.CodeBadRequest, "invalid user id")
	}

	if err := h.service.Delete(r.Context(), id, userID); err != nil {
		return err
	}

	httpio.RespondNoContent(w)
	return nil
}

// --- Session Handler DeleteAllExcept DELETE  ---

func (h *Handler) DeleteAllExcept(w http.ResponseWriter, r *http.Request) error {
	userIDStr := r.PathValue("userId")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return apperr.New(apperr.CodeBadRequest, "invalid user id")
	}

	exceptIDStr := r.URL.Query().Get("exceptId")
	exceptID, err := uuid.Parse(exceptIDStr)
	if err != nil {
		return apperr.New(apperr.CodeBadRequest, "invalid exception session id")
	}

	if err := h.service.DeleteAllExcept(r.Context(), userID, exceptID); err != nil {
		return err
	}

	httpio.RespondNoContent(w)
	return nil
}
