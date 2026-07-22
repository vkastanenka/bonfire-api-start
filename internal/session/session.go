package session

import (
	"errors"
	"net/netip"
	"time"

	"github.com/google/uuid"
)

var (
	ErrSessionExpired = errors.New("session has expired")
	ErrSessionRevoked = errors.New("session has been revoked")
)

type Session struct {
	id               uuid.UUID
	userID           uuid.UUID
	refreshTokenHash RefreshTokenHash
	clientIP         netip.Addr
	userAgent        string
	os               string
	browser          string
	expiresAt        time.Time
	lastSeenAt       time.Time
	revokedAt        *time.Time
	createdAt        time.Time
	updatedAt        time.Time
}

// New constructs a brand new active session aggregate root.
func New(
	userID uuid.UUID,
	tokenHash RefreshTokenHash,
	expiresAt time.Time,
	clientIP netip.Addr,
	userAgent, os, browser string,
) (*Session, error) {
	if userID == uuid.Nil {
		return nil, errors.New("user ID cannot be nil")
	}
	if time.Now().After(expiresAt) {
		return nil, errors.New("expiration time must be in the future")
	}

	now := time.Now().UTC()
	return &Session{
		id:               uuid.New(), // Or allow db/uuidv7 generation
		userID:           userID,
		refreshTokenHash: tokenHash,
		clientIP:         clientIP,
		userAgent:        userAgent,
		os:               os,
		browser:          browser,
		expiresAt:        expiresAt,
		lastSeenAt:       now,
		createdAt:        now,
		updatedAt:        now,
	}, nil
}

// Reconstitute restores an existing session aggregate from storage.
func Reconstitute(
	id, userID uuid.UUID,
	tokenHash RefreshTokenHash,
	clientIP netip.Addr,
	userAgent, os, browser string,
	expiresAt, lastSeenAt, createdAt, updatedAt time.Time,
	revokedAt *time.Time,
) *Session {
	return &Session{
		id:               id,
		userID:           userID,
		refreshTokenHash: tokenHash,
		clientIP:         clientIP,
		userAgent:        userAgent,
		os:               os,
		browser:          browser,
		expiresAt:        expiresAt,
		lastSeenAt:       lastSeenAt,
		revokedAt:        revokedAt,
		createdAt:        createdAt,
		updatedAt:        updatedAt,
	}
}

// --- Domain Business Logic & Mutations ---

func (s *Session) RotateToken(newHash RefreshTokenHash, newExpiresAt time.Time) error {
	if s.IsRevoked() {
		return ErrSessionRevoked
	}
	if s.IsExpired() {
		return ErrSessionExpired
	}
	s.refreshTokenHash = newHash
	s.expiresAt = newExpiresAt
	s.lastSeenAt = time.Now().UTC()
	s.updatedAt = time.Now().UTC()
	return nil
}

func (s *Session) TouchLastSeen() error {
	if s.IsRevoked() {
		return ErrSessionRevoked
	}
	s.lastSeenAt = time.Now().UTC()
	return nil
}

func (s *Session) Revoke() {
	if s.revokedAt == nil {
		now := time.Now().UTC()
		s.revokedAt = &now
		s.updatedAt = now
	}
}

func (s *Session) IsRevoked() bool {
	return s.revokedAt != nil
}

func (s *Session) IsExpired() bool {
	return time.Now().After(s.expiresAt)
}

func (s Session) IsValid() bool { return !s.IsRevoked() && !s.IsExpired() }

// Getters
func (s *Session) ID() uuid.UUID                      { return s.id }
func (s *Session) UserID() uuid.UUID                  { return s.userID }
func (s *Session) RefreshTokenHash() RefreshTokenHash { return s.refreshTokenHash }
func (s *Session) ClientIP() netip.Addr               { return s.clientIP }
func (s *Session) UserAgent() string                  { return s.userAgent }
func (s *Session) OS() string                         { return s.os }
func (s *Session) Browser() string                    { return s.browser }
func (s *Session) ExpiresAt() time.Time               { return s.expiresAt }
func (s *Session) LastSeenAt() time.Time              { return s.lastSeenAt }
func (s *Session) RevokedAt() *time.Time              { return s.revokedAt }
func (s *Session) CreatedAt() time.Time               { return s.createdAt }
func (s *Session) UpdatedAt() time.Time               { return s.updatedAt }
