package session

import (
	"bytes"
	"cmp"
	"net/netip"
	"slices"
	"time"

	"github.com/google/uuid"
)

const (
	maxSessions             = 10
	listValidByUserIDLimit  = maxSessions
	deleteBatchExpiredLimit = 100
)

type Session struct {
	ID               uuid.UUID
	UserID           uuid.UUID
	RefreshTokenHash string
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

func Reconstitute(
	id uuid.UUID,
	userID uuid.UUID,
	refreshTokenHash string,
	clientIP netip.Addr,
	userAgent string,
	os string,
	client string,
	expiresAt,
	lastSeenAt time.Time,
	revokedAt *time.Time,
	createdAt,
	updatedAt time.Time,
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

func New(
	userID uuid.UUID,
	refreshTokenHash string,
	clientIP netip.Addr,
	userAgent string,
	os string,
	client string,
	expiresAt,
	now time.Time,
) (*Session, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}

	return Reconstitute(
		id,
		userID,
		refreshTokenHash,
		clientIP,
		userAgent,
		os,
		client,
		expiresAt,
		now,
		nil,
		now,
		now,
	), nil
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

func sort(sessions []*Session) {
	slices.SortFunc(sessions, func(a, b *Session) int {
		return cmp.Or(
			b.LastSeenAt.Compare(a.LastSeenAt),
			b.CreatedAt.Compare(a.CreatedAt),
			bytes.Compare(a.ID[:], b.ID[:]),
		)
	})
}
