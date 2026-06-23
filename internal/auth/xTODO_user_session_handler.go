package auth

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/httpio"
	"net/http"
	"net/netip"

	"github.com/google/uuid"
)

// ==========================================
// META
// ==========================================

// Count GET
func (h *Handler) CountUserSessions(w http.ResponseWriter, r *http.Request) error {
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

// Create POST
func (h *Handler) CreateUserSession(w http.ResponseWriter, r *http.Request) error {
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

	view, err := h.service.CreateUserSession(r.Context(), CreateUserSessionParams{
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

	httpio.RespondCreated(w, r, view, "")
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

	views, err := h.service.ListActiveUserSessionByUserID(r.Context(), userID)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, views, "")
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

	view, err := h.service.GetUserSessionByID(r.Context(), id)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, view, "")
	return nil
}

// GetByRefreshToken GET
func (h *Handler) GetByRefreshToken(w http.ResponseWriter, r *http.Request) error {
	token := r.URL.Query().Get("token")
	if token == "" {
		return apperr.New(apperr.CodeBadRequest, "refresh token query parameter required")
	}

	view, err := h.service.GetUserSessionByRefreshToken(r.Context(), token)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, view, "")
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

	view, err := h.service.UpdateUserSessionRefreshToken(r.Context(), UpdateRefreshTokenParams{
		ID:           id,
		RefreshToken: reqData.RefreshToken,
		ExpiresAt:    reqData.ExpiresAt,
	})
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, view, "")
	return nil
}

// UpdateLastSeen PATCH
func (h *Handler) UpdateLastSeen(w http.ResponseWriter, r *http.Request) error {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return apperr.New(apperr.CodeBadRequest, "invalid session id")
	}

	view, err := h.service.UpdateUserSessionLastSeen(r.Context(), id)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, view, "ok")
	return nil
}

// MarkBlocked POST/PATCH
func (h *Handler) MarkBlocked(w http.ResponseWriter, r *http.Request) error {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return apperr.New(apperr.CodeBadRequest, "invalid session id")
	}

	view, err := h.service.MarkUserSessionBlocked(r.Context(), id)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, view, "ok")
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

	if err := h.service.DeleteUserSession(r.Context(), id, userID); err != nil {
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

	if err := h.service.DeleteAllUserSessionExcept(r.Context(), userID, exceptID); err != nil {
		return err
	}

	httpio.RespondNoContent(w)
	return nil
}

// PurgeExpired POST
func (h *Handler) PurgeExpired(w http.ResponseWriter, r *http.Request) error {
	if err := h.service.PurgeExpiredUserSession(r.Context()); err != nil {
		return err
	}

	httpio.RespondOK(w, r, PurgeRes{Message: "expired sessions purged successfully"}, "ok")
	return nil
}
