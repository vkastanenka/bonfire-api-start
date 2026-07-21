package session

import (
	"bonfire-api/internal/pkg/ptr"
	"time"

	"github.com/google/uuid"
)

type AuthView struct {
	ID               uuid.UUID  `json:"id"`
	UserID           uuid.UUID  `json:"user_id"`
	RefreshTokenHash []byte     `json:"refresh_token_hash"`
	ExpiresAt        time.Time  `json:"expires_at"`
	RevokedAt        *time.Time `json:"revoked_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

func ToAuthView(s Session) AuthView {
	return AuthView{
		ID:               s.ID,
		UserID:           s.UserID,
		RefreshTokenHash: s.RefreshTokenHash,
		ExpiresAt:        s.ExpiresAt,
		RevokedAt:        ptr.Map(s.RevokedAt),
		CreatedAt:        s.CreatedAt,
		UpdatedAt:        s.UpdatedAt,
	}
}

func FromAuthView(v AuthView) Session {
	return Session{
		ID:               v.ID,
		UserID:           v.UserID,
		RefreshTokenHash: v.RefreshTokenHash,
		ExpiresAt:        v.ExpiresAt,
		RevokedAt:        v.RevokedAt,
		CreatedAt:        v.CreatedAt,
		UpdatedAt:        v.UpdatedAt,
	}
}

type PublicView struct {
	ID         uuid.UUID `json:"id"`
	UserID     uuid.UUID `json:"user_id"`
	LastSeenAt time.Time `json:"last_seen_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	ClientIP   string    `json:"client_ip"`
	UserAgent  string    `json:"user_agent"`
	OS         string    `json:"os"`
	Browser    string    `json:"browser"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func ToPublicView(s Session) PublicView {
	return PublicView{
		ID:         s.ID,
		UserID:     s.UserID,
		LastSeenAt: s.LastSeenAt,
		ExpiresAt:  s.ExpiresAt,
		ClientIP:   s.ClientIP,
		UserAgent:  s.UserAgent,
		OS:         s.OS,
		Browser:    s.Browser,
		CreatedAt:  s.CreatedAt,
		UpdatedAt:  s.UpdatedAt,
	}
}
