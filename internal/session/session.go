package session

import (
	"bonfire-api/internal/pkg/ptr"
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
		s.RevokedAt = ptr.To(row.RevokedAt.Time)
	}
	if row.ClientIP.IsValid() {
		s.ClientIP = row.ClientIP.String()
	}

	return s
}

type View struct {
	ID         uuid.UUID  `json:"id"`
	UserID     uuid.UUID  `json:"user_id"`
	LastSeenAt time.Time  `json:"last_seen_at"`
	ExpiresAt  time.Time  `json:"expires_at"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
	ClientIP   string     `json:"client_ip"`
	UserAgent  string     `json:"user_agent"`
	OS         string     `json:"os"`
	Browser    string     `json:"browser"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

func ToView(s Session) View {
	return View{
		ID:         s.ID,
		UserID:     s.UserID,
		LastSeenAt: s.LastSeenAt,
		ExpiresAt:  s.ExpiresAt,
		RevokedAt:  ptr.Map(s.RevokedAt),
		ClientIP:   s.ClientIP,
		UserAgent:  s.UserAgent,
		OS:         s.OS,
		Browser:    s.Browser,
		CreatedAt:  s.CreatedAt,
		UpdatedAt:  s.UpdatedAt,
	}
}

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
