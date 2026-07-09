package session

import (
	"time"

	"bonfire-api/internal/repository"

	"github.com/google/uuid"
)

type View struct {
	ID         uuid.UUID  `json:"id"`
	UserID     uuid.UUID  `json:"user_id"`
	LastSeenAt time.Time  `json:"last_seen_at"`
	ExpiresAt  time.Time  `json:"expires_at"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
	ClientIP   string     `json:"client_ip"`
	UserAgent  string     `json:"user_agent"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	IsRevoked  bool       `json:"is_revoked"`
	IsExpired  bool       `json:"is_expired"`
}

func NewView(row repository.Session) View {
	view := View{
		ID:         uuid.UUID(row.ID.Bytes),
		UserID:     uuid.UUID(row.UserID.Bytes),
		LastSeenAt: row.LastSeenAt.Time,
		ExpiresAt:  row.ExpiresAt.Time,
		UserAgent:  row.UserAgent,
		CreatedAt:  row.CreatedAt.Time,
		UpdatedAt:  row.UpdatedAt.Time,
		IsRevoked:  row.RevokedAt.Valid,
		IsExpired:  time.Now().After(row.ExpiresAt.Time),
	}

	if row.RevokedAt.Valid {
		view.RevokedAt = &row.RevokedAt.Time
	}

	if row.ClientIP.IsValid() {
		view.ClientIP = row.ClientIP.String()
	} else {
		view.ClientIP = "Unknown"
	}

	return view
}

type AuthView struct {
	ID               uuid.UUID  `json:"id"`
	UserID           uuid.UUID  `json:"user_id"`
	RefreshTokenHash []byte     `json:"refresh_token_hash"`
	ExpiresAt        time.Time  `json:"expires_at"`
	RevokedAt        *time.Time `json:"revoked_at,omitempty"`
}

func NewAuthView(row repository.Session) AuthView {
	authView := AuthView{
		ID:               uuid.UUID(row.ID.Bytes),
		UserID:           uuid.UUID(row.UserID.Bytes),
		RefreshTokenHash: row.RefreshTokenHash,
		ExpiresAt:        row.ExpiresAt.Time,
	}

	if row.RevokedAt.Valid {
		authView.RevokedAt = &row.RevokedAt.Time
	}

	return authView
}

func (a AuthView) IsRevoked() bool {
	return a.RevokedAt != nil
}

func (a AuthView) IsExpired() bool {
	return time.Now().After(a.ExpiresAt)
}

func (a AuthView) IsValid() bool {
	return !a.IsRevoked() && !a.IsExpired()
}
