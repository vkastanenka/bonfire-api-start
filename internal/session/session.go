package session

import (
	"net/netip"
	"time"

	"github.com/google/uuid"
)

type Session struct {
	ID               uuid.UUID
	UserID           uuid.UUID
	RefreshTokenHash RefreshTokenHash
	LastSeenAt       time.Time
	ExpiresAt        time.Time
	RevokedAt        *time.Time
	ClientIP         netip.Addr
	UserAgent        string
	OS               string
	Browser          string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (s Session) IsRevoked() bool { return s.RevokedAt != nil }
func (s Session) IsExpired() bool { return time.Now().After(s.ExpiresAt) }
func (s Session) IsValid() bool   { return !s.IsRevoked() && !s.IsExpired() }
