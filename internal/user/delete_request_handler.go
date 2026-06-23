package user

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/httpio"
	"net/http"

	"github.com/google/uuid"
)

// ==========================================
// META
// ==========================================

// Count GET
func (h *Handler) CountDeleteRequests(w http.ResponseWriter, r *http.Request) error {
	count, err := h.service.CountDeleteRequests(r.Context())
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
func (h *Handler) CreateDeleteRequest(w http.ResponseWriter, r *http.Request) error {
	reqData, err := httpio.BindJSON[CreateDeleteRequestReq](w, r, h.validator)
	if err != nil {
		return err
	}

	userID, err := uuid.Parse(reqData.UserID)
	if err != nil {
		return apperr.New(apperr.CodeBadRequest, ErrInvalidID)
	}

	row, err := h.service.CreateDeleteRequest(r.Context(), CreateDeleteRequestParams{
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
func (h *Handler) ListDeleteRequestsDue(w http.ResponseWriter, r *http.Request) error {
	rows, err := h.service.ListDeleteRequestDue(r.Context())
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, rows, ListDeleteRequestsDueOK)
	return nil
}

// ==========================================
// GET
// ==========================================

// GetByUserID GET
func (h *Handler) GetDeleteRequestByUserID(w http.ResponseWriter, r *http.Request) error {
	userIDStr := r.PathValue("userId")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return apperr.New(apperr.CodeBadRequest, ErrInvalidID)
	}

	row, err := h.service.GetDeleteRequestByUserID(r.Context(), userID)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, row, GetDeleteRequestByUserIDOK)
	return nil
}

// ==========================================
// DELETE
// ==========================================

// DeleteByUserID DELETE
func (h *Handler) DeleteDeleteRequestByUserID(w http.ResponseWriter, r *http.Request) error {
	userIDStr := r.PathValue("userId")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return apperr.New(apperr.CodeBadRequest, ErrInvalidID)
	}

	if err := h.service.DeleteDeleteRequestByUserID(r.Context(), userID); err != nil {
		return err
	}

	httpio.RespondNoContent(w)
	return nil
}
