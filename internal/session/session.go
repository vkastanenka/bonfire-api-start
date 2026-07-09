package session

import (
	"bonfire-api/internal/repository"
	"time"

	"github.com/google/uuid"
)

type Session struct {
	ID               uuid.UUID
	UserID           uuid.UUID
	RefreshTokenHash []byte
	LastSeenAt       time.Time
	ExpiresAt        time.Time
	RevokedAt        *time.Time
	ClientIP         string
	UserAgent        string
	OS               string
	Browser          string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (s *Session) IsRevoked() bool { return s.RevokedAt != nil }
func (s *Session) IsExpired() bool { return time.Now().After(s.ExpiresAt) }
func (s *Session) IsValid() bool   { return !s.IsRevoked() && !s.IsExpired() }

func FromRepository(row repository.Session) Session {
	s := Session{
		ID:               uuid.UUID(row.ID.Bytes),
		UserID:           uuid.UUID(row.UserID.Bytes),
		RefreshTokenHash: row.RefreshTokenHash,
		LastSeenAt:       row.LastSeenAt.Time,
		ExpiresAt:        row.ExpiresAt.Time,
		UserAgent:        row.UserAgent,
		OS:               row.OS,
		Browser:          row.Browser,
		CreatedAt:        row.CreatedAt.Time,
		UpdatedAt:        row.UpdatedAt.Time,
	}
	if row.RevokedAt.Valid {
		s.RevokedAt = &row.RevokedAt.Time
	}
	if row.ClientIP.IsValid() {
		s.ClientIP = row.ClientIP.String()
	}
	return s
}

type AuthView struct {
	ID               uuid.UUID  `json:"id"`
	UserID           uuid.UUID  `json:"user_id"`
	RefreshTokenHash []byte     `json:"refresh_token_hash"`
	ExpiresAt        time.Time  `json:"expires_at"`
	RevokedAt        *time.Time `json:"revoked_at"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

func (a *AuthView) IsRevoked() bool { return a.RevokedAt != nil }
func (a *AuthView) IsExpired() bool { return time.Now().After(a.ExpiresAt) }
func (a *AuthView) IsValid() bool   { return !a.IsRevoked() && !a.IsExpired() }

func (s *Session) ToAuthView() AuthView {
	return AuthView{
		ID:               s.ID,
		UserID:           s.UserID,
		RefreshTokenHash: s.RefreshTokenHash,
		ExpiresAt:        s.ExpiresAt,
		RevokedAt:        s.RevokedAt,
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
}

func (s *Session) ToPublicView() PublicView {
	return PublicView{
		ID:         s.ID,
		UserID:     s.UserID,
		LastSeenAt: s.LastSeenAt,
		ExpiresAt:  s.ExpiresAt,
		ClientIP:   s.ClientIP,
		UserAgent:  s.UserAgent,
		OS:         s.OS,
		Browser:    s.Browser,
	}
}
