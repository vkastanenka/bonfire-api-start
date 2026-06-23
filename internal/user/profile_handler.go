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

func (h *Handler) CountProfiles(w http.ResponseWriter, r *http.Request) error {
	count, err := h.service.CountProfiles(r.Context())
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, CountRes{Count: count}, CountOK)
	return nil
}

// ==========================================
// CREATE
// ==========================================

func (h *Handler) CreateProfile(w http.ResponseWriter, r *http.Request) error {
	reqData, err := httpio.BindJSON[CreateProfileReq](w, r, h.validator)
	if err != nil {
		return err
	}

	userID, err := uuid.Parse(reqData.UserID)
	if err != nil {
		return apperr.New(apperr.CodeBadRequest, ErrInvalidID)
	}

	view, err := h.service.CreateProfile(r.Context(), CreateProfileParams{
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

func (h *Handler) GetProfileByUserID(w http.ResponseWriter, r *http.Request) error {
	userIDStr := r.PathValue("userId")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return apperr.New(apperr.CodeBadRequest, ErrInvalidID)
	}

	view, err := h.service.GetProfileByUserID(r.Context(), userID)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, view, GetProfileByUserIDOK)
	return nil
}

// ==========================================
// UPDATE
// ==========================================

func (h *Handler) UpdateProfileDisplayName(w http.ResponseWriter, r *http.Request) error {
	userIDStr := r.PathValue("userId")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return apperr.New(apperr.CodeBadRequest, ErrInvalidID)
	}

	reqData, err := httpio.BindJSON[UpdateProfileDisplayNameReq](w, r, h.validator)
	if err != nil {
		return err
	}

	view, err := h.service.UpdateProfileDisplayName(r.Context(), UpdateProfileDisplayNameParams{
		UserID:      userID,
		DisplayName: reqData.DisplayName,
	})
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, view, UpdateProfileDisplayNameOK)
	return nil
}

// ==========================================
// DELETE
// ==========================================

func (h *Handler) DeleteProfileByUserID(w http.ResponseWriter, r *http.Request) error {
	userIDStr := r.PathValue("userId")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return apperr.New(apperr.CodeBadRequest, ErrInvalidID)
	}

	if err := h.service.DeleteProfileByUserID(r.Context(), userID); err != nil {
		return err
	}

	httpio.RespondNoContent(w)
	return nil
}
