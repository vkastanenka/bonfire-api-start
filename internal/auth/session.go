package auth

import (
	"bytes"
	"cmp"
	"net/netip"
	"slices"
	"time"

	"github.com/google/uuid"
)

type Session struct {
	ID               uuid.UUID
	UserID           uuid.UUID
	RefreshTokenHash [32]byte
	ClientIP         netip.Addr
	UserAgent        string
	OS               string
	Client           string
	ExpiresAt        time.Time
	LastSeenAt       time.Time
	RevokedAt        *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func New(
	id, userID uuid.UUID,
	refreshTokenHash [32]byte,
	clientIP netip.Addr,
	userAgent, os, client string,
	expiresAt, now time.Time,
) *Session {
	return &Session{
		ID:               id,
		UserID:           userID,
		RefreshTokenHash: refreshTokenHash,
		ClientIP:         clientIP,
		UserAgent:        userAgent,
		OS:               os,
		Client:           client,
		ExpiresAt:        expiresAt,
		LastSeenAt:       now,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

func Reconstitute(
	id, userID uuid.UUID,
	refreshTokenHash [32]byte,
	clientIP netip.Addr,
	userAgent, os, client string,
	expiresAt, lastSeenAt time.Time,
	revokedAt *time.Time,
	createdAt, updatedAt time.Time,
) *Session {
	return &Session{
		ID:               id,
		UserID:           userID,
		RefreshTokenHash: refreshTokenHash,
		ClientIP:         clientIP,
		UserAgent:        userAgent,
		OS:               os,
		Client:           client,
		ExpiresAt:        expiresAt,
		LastSeenAt:       lastSeenAt,
		RevokedAt:        revokedAt,
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAt,
	}
}

func (s *Session) IsRevoked() bool {
	return s.RevokedAt != nil
}

func (s *Session) IsExpired(now time.Time) bool {
	return !s.ExpiresAt.After(now)
}

func (s *Session) IsValid(now time.Time) bool {
	return !s.IsRevoked() && !s.IsExpired(now)
}

func sortSessions(sessions []*Session) {
	slices.SortFunc(sessions, func(a, b *Session) int {
		return cmp.Or(
			b.LastSeenAt.Compare(a.LastSeenAt),
			bytes.Compare(b.ID[:], a.ID[:]),
		)
	})
}
