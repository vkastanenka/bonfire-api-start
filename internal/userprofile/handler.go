package userprofile

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
		return apperr.New(apperr.CodeBadRequest, ErrInvalidID)
	}

	view, err := h.service.Create(r.Context(), CreateParams{
		UserID:      userID,
		DisplayName: reqData.DisplayName,
	})
	if err != nil {
		return err
	}

	httpio.RespondCreated(w, r, view, CreateOK)
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
		return apperr.New(apperr.CodeBadRequest, ErrInvalidID)
	}

	view, err := h.service.GetByUserID(r.Context(), userID)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, view, GetByUserIDOK)
	return nil
}

// ==========================================
// UPDATE
// ==========================================

// UpdateDisplayName PATCH/PUT
func (h *Handler) UpdateDisplayName(w http.ResponseWriter, r *http.Request) error {
	userIDStr := r.PathValue("userId")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return apperr.New(apperr.CodeBadRequest, ErrInvalidID)
	}

	reqData, err := httpio.BindJSON[UpdateDisplayNameReq](w, r, h.validator)
	if err != nil {
		return err
	}

	view, err := h.service.UpdateDisplayName(r.Context(), UpdateDisplayNameParams{
		UserID:      userID,
		DisplayName: reqData.DisplayName,
	})
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, view, UpdateDisplayNameOK)
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
		return apperr.New(apperr.CodeBadRequest, ErrInvalidID)
	}

	if err := h.service.DeleteByUserID(r.Context(), userID); err != nil {
		return err
	}

	httpio.RespondNoContent(w)
	return nil
}
