package userdeleterequest

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/validator"
	"net/http"

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

// ==========================================
// CREATE
// ==========================================

// Create POST
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) error {
	reqData, err := httpio.BindJSON[CreateReq](w, r, h.validator)
	if err != nil {
		return err
	}

	userID, err := uuid.Parse(reqData.UserID)
	if err != nil {
		return apperr.New(apperr.CodeBadRequest, "invalid user id")
	}

	row, err := h.service.Create(r.Context(), CreateParams{
		UserID:      userID,
		ScheduledAt: reqData.ScheduledAt,
	})
	if err != nil {
		return err
	}

	httpio.RespondCreated(w, r, row, CreateOK)
	return nil
}

// ==========================================
// LIST
// ==========================================

// ListDue GET
func (h *Handler) ListDue(w http.ResponseWriter, r *http.Request) error {
	rows, err := h.service.ListDue(r.Context())
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, rows, ListDueOK)
	return nil
}

// ==========================================
// GET
// ==========================================

// GetByUserID GET
func (h *Handler) GetByUserID(w http.ResponseWriter, r *http.Request) error {
	userIDStr := r.PathValue("userId")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return apperr.New(apperr.CodeBadRequest, "invalid user id")
	}

	row, err := h.service.GetByUserID(r.Context(), userID)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, row, GetByUserIDOK)
	return nil
}

// ==========================================
// DELETE
// ==========================================

// DeleteByUserID DELETE
func (h *Handler) DeleteByUserID(w http.ResponseWriter, r *http.Request) error {
	userIDStr := r.PathValue("userId")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return apperr.New(apperr.CodeBadRequest, "invalid user id")
	}

	if err := h.service.DeleteByUserID(r.Context(), userID); err != nil {
		return err
	}

	httpio.RespondNoContent(w)
	return nil
}
