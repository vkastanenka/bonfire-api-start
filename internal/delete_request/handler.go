package delete_request

import (
	"net/http"

	"bonfire-api/internal/apperr"
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/validator"
)

// --- delete_request handler ---

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

// --- delete_request handler Count ---

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
// CREATE
// ==========================================

// --- delete_request handler Create ---

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) error {
	userID, err := httpio.GetCtxUserID(r.Context())
	if err != nil {
		return apperr.NewUnauthorized(err)
	}

	row, err := h.service.Create(r.Context(), userID)
	if err != nil {
		return err
	}

	httpio.RespondCreated(w, r, row, "")
	return nil
}

// ==========================================
// LIST
// ==========================================

// --- delete_request handler ListDue ---

func (h *Handler) ListDue(w http.ResponseWriter, r *http.Request) error {
	views, err := h.service.ListDue(r.Context())
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, views, "")
	return nil
}

// ==========================================
// GET
// ==========================================

// --- delete_request handler GetByUserID  ---

func (h *Handler) GetByUserID(w http.ResponseWriter, r *http.Request) error {
	userID, err := httpio.GetCtxUserID(r.Context())
	if err != nil {
		return apperr.NewUnauthorized(err)
	}

	view, err := h.service.GetByUserID(r.Context(), userID)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, view, "")
	return nil
}

// ==========================================
// DELETE
// ==========================================

// --- delete_request handler Delete  ---

func (h *Handler) DeleteByUserID(w http.ResponseWriter, r *http.Request) error {
	userID, err := httpio.GetCtxUserID(r.Context())
	if err != nil {
		return apperr.NewUnauthorized(err)
	}

	if err := h.service.DeleteByUserID(r.Context(), userID); err != nil {
		return err
	}

	httpio.RespondNoContent(w)
	return nil
}
