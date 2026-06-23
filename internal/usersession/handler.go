package userprofile

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/validator"
	"net/http"
	"net/netip"

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

	httpio.RespondOK(w, r, CountRes{Count: count}, CountBytesOK)
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

	// Extract Client IP address safely from RemoteAddr
	var clientIP netip.Addr
	if addrPort, err := netip.ParseAddrPort(r.RemoteAddr); err == nil {
		clientIP = addrPort.Addr()
	} else if addr, err := netip.ParseAddr(r.RemoteAddr); err == nil {
		clientIP = addr
	} else {
		return apperr.New(apperr.CodeBadRequest, "invalid remote client ip address")
	}

	// Automatically assign a new chronological ID (UUIDv7) for tracking
	sessionID, err := uuid.NewV7()
	if err != nil {
		return err
	}

	view, err := h.service.Create(r.Context(), CreateParams{
		ID:           sessionID,
		UserID:       userID,
		RefreshToken: reqData.RefreshToken,
		UserAgent:    r.UserAgent(),
		ClientIP:     clientIP,
		IsBlocked:    false,
		ExpiresAt:    reqData.ExpiresAt,
	})
	if err != nil {
		return err
	}

	httpio.RespondCreated(w, r, view, CreateOK)
	return nil
}

// ==========================================
// LIST
// ==========================================

// ListActiveByUserID GET
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

	httpio.RespondOK(w, r, views, ListActiveOK)
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
		return apperr.New(apperr.CodeBadRequest, "invalid session id")
	}

	view, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, view, GetByIDOK)
	return nil
}

// GetByRefreshToken GET
func (h *Handler) GetByRefreshToken(w http.ResponseWriter, r *http.Request) error {
	token := r.URL.Query().Get("token")
	if token == "" {
		return apperr.New(apperr.CodeBadRequest, "refresh token query parameter required")
	}

	view, err := h.service.GetByRefreshToken(r.Context(), token)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, view, GetByRefreshTokenOK)
	return nil
}

// ==========================================
// UPDATE
// ==========================================

// UpdateRefreshToken PUT/PATCH
func (h *Handler) UpdateRefreshToken(w http.ResponseWriter, r *http.Request) error {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return apperr.New(apperr.CodeBadRequest, "invalid session id")
	}

	reqData, err := httpio.BindJSON[UpdateRefreshTokenReq](w, r, h.validator)
	if err != nil {
		return err
	}

	view, err := h.service.UpdateRefreshToken(r.Context(), UpdateRefreshTokenParams{
		ID:           id,
		RefreshToken: reqData.RefreshToken,
		ExpiresAt:    reqData.ExpiresAt,
	})
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, view, UpdateRefreshTokenOK)
	return nil
}

// UpdateLastSeen PATCH
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

	httpio.RespondOK(w, r, view, UpdateLastSeenOK)
	return nil
}

// MarkBlocked POST/PATCH
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

	httpio.RespondOK(w, r, view, MarkBlockedOK)
	return nil
}

// ==========================================
// DELETE
// ==========================================

// Delete DELETE
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

// DeleteAllExcept DELETE
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

// PurgeExpired POST
func (h *Handler) PurgeExpired(w http.ResponseWriter, r *http.Request) error {
	if err := h.service.PurgeExpired(r.Context()); err != nil {
		return err
	}

	httpio.RespondOK(w, r, PurgeRes{Message: "expired sessions purged successfully"}, PurgeExpiredOK)
	return nil
}
