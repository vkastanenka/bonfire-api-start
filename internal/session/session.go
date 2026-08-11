package session

import (
	"bonfire-api/internal/errs"
	"bonfire-api/internal/fields"
	"time"
)

var (
	ErrSessionExpired = errs.Unauthenticated("Session has expired.").
				Reason("SESSION_EXPIRED")
	ErrSessionRevoked = errs.Unauthenticated("Session has been revoked.").
				Reason("SESSION_REVOKED")
	ErrSessionInvalid = errs.Unauthenticated("Session is invalid.").
				Reason("SESSION_INVALID")
)

type Session struct {
	id               fields.ID
	userID           fields.ID
	refreshTokenHash RefreshTokenHash
	clientIP         ClientIP
	userAgent        UserAgent
	os               OS
	client           Client
	expiresAt        ExpiresAt
	lastSeenAt       fields.Timestamp
	revokedAt        fields.Timestamp
	createdAt        fields.Timestamp
	updatedAt        fields.Timestamp
}

// ============================================================================
// Getters
// ============================================================================

func (s *Session) ID() fields.ID                      { return s.id }
func (s *Session) UserID() fields.ID                  { return s.userID }
func (s *Session) RefreshTokenHash() RefreshTokenHash { return s.refreshTokenHash }
func (s *Session) ClientIP() ClientIP                 { return s.clientIP }
func (s *Session) UserAgent() UserAgent               { return s.userAgent }
func (s *Session) OS() OS                             { return s.os }
func (s *Session) Client() Client                     { return s.client }
func (s *Session) ExpiresAt() ExpiresAt               { return s.expiresAt }
func (s *Session) LastSeenAt() fields.Timestamp       { return s.lastSeenAt }
func (s *Session) RevokedAt() fields.Timestamp        { return s.revokedAt }
func (s *Session) CreatedAt() fields.Timestamp        { return s.createdAt }
func (s *Session) UpdatedAt() fields.Timestamp        { return s.updatedAt }

// ============================================================================
// Meta
// ============================================================================

func (s *Session) IsRevoked() bool { return s.revokedAt.StringPtr() != nil }
func (s *Session) IsExpired() bool { return time.Now().After(s.expiresAt.Time()) }
func (s Session) IsValid() bool    { return !s.IsRevoked() && !s.IsExpired() }

// ============================================================================
// Mappers
// ============================================================================

func New(
	id fields.ID,
	userID fields.ID,
	refreshTokenHash RefreshTokenHash,
	clientIP ClientIP,
	userAgent UserAgent,
	os OS,
	client Client,
	expiresAt ExpiresAt,
	now fields.Timestamp,
) (*Session, error) {
	return &Session{
		id:               id,
		userID:           userID,
		refreshTokenHash: refreshTokenHash,
		clientIP:         clientIP,
		userAgent:        userAgent,
		os:               os,
		client:           client,
		expiresAt:        expiresAt,
		lastSeenAt:       now,
		createdAt:        now,
		updatedAt:        now,
	}, nil
}

func Reconstitute(
	id fields.ID,
	userID fields.ID,
	refreshTokenHash RefreshTokenHash,
	clientIP ClientIP,
	userAgent UserAgent,
	os OS,
	client Client,
	expiresAt ExpiresAt,
	lastSeenAt, revokedAt, createdAt, updatedAt fields.Timestamp,
) *Session {
	return &Session{
		id:               id,
		userID:           userID,
		refreshTokenHash: refreshTokenHash,
		clientIP:         clientIP,
		userAgent:        userAgent,
		os:               os,
		client:           client,
		expiresAt:        expiresAt,
		lastSeenAt:       lastSeenAt,
		revokedAt:        revokedAt,
		createdAt:        createdAt,
		updatedAt:        updatedAt,
	}
}

// ============================================================================
// Mutations
// ============================================================================

func (s *Session) RotateToken(
	newHash RefreshTokenHash,
	newExpiresAt ExpiresAt,
	newClientIP ClientIP,
	newUserAgent UserAgent,
	now fields.Timestamp,
) error {
	s.refreshTokenHash = newHash
	s.expiresAt = newExpiresAt
	s.clientIP = newClientIP
	s.userAgent = newUserAgent
	s.TouchLastSeen(now)
	return nil
}

func (s *Session) UpdateMetadata(
	clientIP ClientIP,
	userAgent UserAgent,
	os OS,
	client Client,
	now fields.Timestamp,
) error {
	s.clientIP = clientIP
	s.userAgent = userAgent
	s.os = os
	s.client = client
	s.TouchLastSeen(now)
	return nil
}

func (s *Session) ExtendExpiry(newExpiresAt ExpiresAt, now fields.Timestamp) error {
	s.expiresAt = newExpiresAt
	s.TouchLastSeen(now)
	return nil
}

func (s *Session) Revoke(now fields.Timestamp) error {
	s.revokedAt = now
	s.TouchLastSeen(now)
	return nil
}

func (s *Session) TouchLastSeen(now fields.Timestamp) error {
	s.lastSeenAt = now
	s.touch(now)
	return nil
}

func (s *Session) touch(at fields.Timestamp) {
	s.updatedAt = at
}
