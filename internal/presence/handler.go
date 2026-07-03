package presence

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/httpio"
	"net/http"

	"github.com/google/uuid"
)

// --- presence handler ---

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

// --- presence handler Heartbeat ---

func (h *Handler) Heartbeat(w http.ResponseWriter, r *http.Request) error {
	userID, err := httpio.GetCtxUserID(r.Context())
	if err != nil {
		return apperr.NewUnauthorized(err, "")
	}

	if err := h.service.Heartbeat(r.Context(), userID.String()); err != nil {
		return err
	}

	httpio.RespondNoContent(w)
	return nil
}

// --- presence handler UpdateStatus ---

type UpdateStatusReq struct {
	Status Activity `json:"status" validate:"required"`
}

func (h *Handler) UpdateStatus(w http.ResponseWriter, r *http.Request) error {
	userID, err := httpio.GetCtxUserID(r.Context())
	if err != nil {
		return apperr.NewUnauthorized(err, "")
	}

	req, err := httpio.BindJSON[UpdateStatusReq](w, r)
	if err != nil {
		return err
	}

	if !req.Status.Valid() {
		return apperr.NewBadRequest(nil, "invalid activity status")
	}

	if err := h.service.UpdateStatus(
		r.Context(),
		userID.String(),
		req.Status,
	); err != nil {
		return err
	}

	httpio.RespondNoContent(w)
	return nil
}

// --- presence handler GetActivity ---

type GetActivityRes struct {
	Status Activity `json:"status"`
}

func (h *Handler) GetActivity(w http.ResponseWriter, r *http.Request) error {
	userID, err := httpio.GetCtxUserID(r.Context())
	if err != nil {
		return apperr.NewUnauthorized(err, "")
	}

	status, err := h.service.GetActivity(r.Context(), userID.String())
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, GetActivityRes{
		Status: status,
	}, "")
	return nil
}

// --- presence handler GetUserActivity ---

type GetUserActivityPath struct {
	ID uuid.UUID `path:"id" validate:"required"`
}

type GetUserActivityRes struct {
	Status Activity `json:"status"`
}

func (h *Handler) GetUserActivity(w http.ResponseWriter, r *http.Request) error {
	_, err := httpio.GetCtxUserID(r.Context())
	if err != nil {
		return apperr.NewUnauthorized(err, "")
	}

	path, err := httpio.BindPath[GetUserActivityPath](r)
	if err != nil {
		return err
	}

	status, err := h.service.GetActivity(r.Context(), path.ID.String())
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, GetUserActivityRes{
		Status: status,
	}, "")

	return nil
}

// --- presence handler GetBulkActivity ---

type BulkActivityReq struct {
	UserIDs []uuid.UUID `json:"user_ids" validate:"required"`
}

type BulkActivityRes struct {
	Statuses map[string]Activity `json:"statuses"`
}

func (h *Handler) GetBulkActivity(w http.ResponseWriter, r *http.Request) error {
	_, err := httpio.GetCtxUserID(r.Context())
	if err != nil {
		return apperr.NewUnauthorized(err, "")
	}

	req, err := httpio.BindJSON[BulkActivityReq](w, r)
	if err != nil {
		return err
	}

	userIDs := make([]string, len(req.UserIDs))
	for i, id := range req.UserIDs {
		userIDs[i] = id.String()
	}

	statuses, err := h.service.GetBulkActivity(r.Context(), userIDs)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, BulkActivityRes{
		Statuses: statuses,
	}, "")

	return nil
}
