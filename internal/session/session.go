package session

import (
	"time"

	"bonfire-api/internal/repository"

	"github.com/google/uuid"
)

// --- Session View ---

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

func NewView(row repository.Session) View {
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
