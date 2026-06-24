package auth

import (
	"context"
	"net/http"
	"net/netip"
	"time"

	"bonfire-api/internal/apperr"
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// --- USER SESSION TYPES ---

// View
type UserSessionView struct {
	ID           uuid.UUID `json:"id"`
	UserID       uuid.UUID `json:"user_id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	ExpiresAt    time.Time `json:"expires_at"`
	LastSeenAt   time.Time `json:"last_seen_at"`
	RefreshToken string    `json:"refresh_token"`
	IsBlocked    bool      `json:"is_blocked"`
	ClientIP     string    `json:"client_ip"`
	UserAgent    string    `json:"user_agent"`
}

func NewUserSessionView(row repository.UserSession) UserSessionView {
	return UserSessionView{
		ID:           row.ID.Bytes,
		UserID:       row.UserID.Bytes,
		CreatedAt:    row.CreatedAt.Time,
		UpdatedAt:    row.UpdatedAt.Time,
		ExpiresAt:    row.ExpiresAt.Time,
		LastSeenAt:   row.LastSeenAt.Time,
		RefreshToken: row.RefreshToken,
		IsBlocked:    row.IsBlocked,
		ClientIP:     row.ClientIP.String(),
		UserAgent:    row.UserAgent,
	}
}

// Requests & Responses
type CountRes struct {
	Count int64 `json:"count"`
}

type CreateReq struct {
	UserID       string    `json:"user_id" validate:"required,uuid"`
	RefreshToken string    `json:"refresh_token" validate:"required"`
	ExpiresAt    time.Time `json:"expires_at" validate:"required"`
}

type UpdateRefreshTokenReq struct {
	RefreshToken string    `json:"refresh_token" validate:"required"`
	ExpiresAt    time.Time `json:"expires_at" validate:"required"`
}

type PurgeRes struct {
	Message string `json:"message"`
}

// Service Parameters
type CreateUserSessionParams struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	RefreshToken string
	UserAgent    string
	ClientIP     netip.Addr
	IsBlocked    bool
	ExpiresAt    time.Time
}

type UpdateRefreshTokenParams struct {
	ID           uuid.UUID
	RefreshToken string
	ExpiresAt    time.Time
}

// --- USER SESSION HANDLERS ---

// ==========================================
// META
// ==========================================

// CountUserSessions GET
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

// CreateUserSession POST
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

// --- USER SESSION SERVICES ---

// ==========================================
// META
// ==========================================

func (s *Service) Count(ctx context.Context) (int64, error) {
	count, err := s.store.UserSessionCount(ctx)
	if err != nil {
		return 0, apperr.NewDBError(err)
	}
	return count, nil
}

// ==========================================
// CREATE
// ==========================================

func (s *Service) CreateUserSession(ctx context.Context, p CreateUserSessionParams) (UserSessionView, error) {
	row, err := s.store.UserSessionCreate(ctx, repository.UserSessionCreateParams{
		ID:           pgtype.UUID{Bytes: p.ID, Valid: true},
		UserID:       pgtype.UUID{Bytes: p.UserID, Valid: true},
		RefreshToken: p.RefreshToken,
		UserAgent:    p.UserAgent,
		ClientIP:     p.ClientIP,
		IsBlocked:    p.IsBlocked,
		ExpiresAt:    pgtype.Timestamptz{Time: p.ExpiresAt, Valid: true},
	})
	if err != nil {
		return UserSessionView{}, apperr.NewDBError(err)
	}
	return NewUserSessionView(row), nil
}

// ==========================================
// LIST
// ==========================================

func (s *Service) ListActiveUserSessionByUserID(ctx context.Context, userID uuid.UUID) ([]UserSessionView, error) {
	rows, err := s.store.UserSessionListActiveByUserID(ctx, pgtype.UUID{Bytes: userID, Valid: true})
	if err != nil {
		return nil, apperr.NewDBError(err)
	}

	views := make([]UserSessionView, len(rows))
	for i, row := range rows {
		views[i] = NewUserSessionView(row)
	}
	return views, nil
}

// ==========================================
// GET
// ==========================================

func (s *Service) GetUserSessionByID(ctx context.Context, id uuid.UUID) (UserSessionView, error) {
	row, err := s.store.UserSessionGetByID(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return UserSessionView{}, apperr.NewDBError(err)
	}
	return NewUserSessionView(row), nil
}

func (s *Service) GetUserSessionByRefreshToken(ctx context.Context, refreshToken string) (UserSessionView, error) {
	row, err := s.store.UserSessionGetByRefreshToken(ctx, refreshToken)
	if err != nil {
		return UserSessionView{}, apperr.NewDBError(err)
	}
	return NewUserSessionView(row), nil
}

// ==========================================
// UPDATE
// ==========================================

func (s *Service) UpdateUserSessionRefreshToken(ctx context.Context, p UpdateRefreshTokenParams) (UserSessionView, error) {
	row, err := s.store.UserSessionUpdateRefreshToken(ctx, repository.UserSessionUpdateRefreshTokenParams{
		ID:           pgtype.UUID{Bytes: p.ID, Valid: true},
		RefreshToken: p.RefreshToken,
		ExpiresAt:    pgtype.Timestamptz{Time: p.ExpiresAt, Valid: true},
	})
	if err != nil {
		return UserSessionView{}, apperr.NewDBError(err)
	}
	return NewUserSessionView(row), nil
}

func (s *Service) UpdateUserSessionLastSeen(ctx context.Context, id uuid.UUID) (UserSessionView, error) {
	row, err := s.store.UserSessionUpdateLastSeen(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return UserSessionView{}, apperr.NewDBError(err)
	}
	return NewUserSessionView(row), nil
}

func (s *Service) MarkUserSessionBlocked(ctx context.Context, id uuid.UUID) (UserSessionView, error) {
	row, err := s.store.UserSessionMarkBlocked(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return UserSessionView{}, apperr.NewDBError(err)
	}
	return NewUserSessionView(row), nil
}

// ==========================================
// DELETE
// ==========================================

func (s *Service) DeleteUserSession(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	err := s.store.UserSessionDelete(ctx, repository.UserSessionDeleteParams{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		UserID: pgtype.UUID{Bytes: userID, Valid: true},
	})
	if err != nil {
		return apperr.NewDBError(err)
	}
	return nil
}

func (s *Service) DeleteAllUserSessionExcept(ctx context.Context, userID uuid.UUID, exceptID uuid.UUID) error {
	err := s.store.UserSessionDeleteAllExcept(ctx, repository.UserSessionDeleteAllExceptParams{
		UserID: pgtype.UUID{Bytes: userID, Valid: true},
		ID:     pgtype.UUID{Bytes: exceptID, Valid: true},
	})
	if err != nil {
		return apperr.NewDBError(err)
	}
	return nil
}

func (s *Service) PurgeExpiredUserSession(ctx context.Context) error {
	err := s.store.UserSessionPurgeExpired(ctx)
	if err != nil {
		return apperr.NewDBError(err)
	}
	return nil
}
