package userprofile

import (
	"bonfire-api/internal/repository"
	"net/netip"
	"time"

	"github.com/google/uuid"
)

// ==========================================
// HANDLERS
// ==========================================

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

// ==========================================
// SERVICES
// ==========================================

type CreateParams struct {
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

// ==========================================
// VIEW
// ==========================================

type View struct {
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

func NewView(row repository.UserSession) View {
	return View{
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
