package session

import (
	"bonfire-api/internal/fields"
)

const (
	MaxUserSessions                = 10
	UserSessionsBatchLimit         = MaxUserSessions
	DefaultDeleteExpiredBatchLimit = 100
)

type Session struct {
	id               fields.ID
	userID           fields.ID
	refreshTokenHash fields.TokenHash
	clientIP         fields.IP
	userAgent        fields.UserAgent
	os               fields.OS
	client           fields.Client
	expiresAt        fields.Timestamp
	lastSeenAt       fields.Timestamp
	revokedAt        fields.Timestamp
	createdAt        fields.Timestamp
	updatedAt        fields.Timestamp
}

func Reconstitute(
	id fields.ID,
	userID fields.ID,
	refreshTokenHash fields.TokenHash,
	clientIP fields.IP,
	userAgent fields.UserAgent,
	os fields.OS,
	client fields.Client,
	expiresAt,
	lastSeenAt,
	revokedAt,
	createdAt,
	updatedAt fields.Timestamp,
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

func New(
	userID fields.ID,
	refreshTokenHash fields.TokenHash,
	clientIP fields.IP,
	userAgent fields.UserAgent,
	os fields.OS,
	client fields.Client,
	expiresAt,
	now fields.Timestamp,
) (*Session, error) {
	id, err := fields.NewID()
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
		fields.Timestamp{},
		now,
		now,
	), nil
}

func (s *Session) ID() fields.ID                      { return s.id }
func (s *Session) UserID() fields.ID                  { return s.userID }
func (s *Session) RefreshTokenHash() fields.TokenHash { return s.refreshTokenHash }
func (s *Session) ClientIP() fields.IP                { return s.clientIP }
func (s *Session) UserAgent() fields.UserAgent        { return s.userAgent }
func (s *Session) OS() fields.OS                      { return s.os }
func (s *Session) Client() fields.Client              { return s.client }
func (s *Session) ExpiresAt() fields.Timestamp        { return s.expiresAt }
func (s *Session) LastSeenAt() fields.Timestamp       { return s.lastSeenAt }
func (s *Session) RevokedAt() fields.Timestamp        { return s.revokedAt }
func (s *Session) CreatedAt() fields.Timestamp        { return s.createdAt }
func (s *Session) UpdatedAt() fields.Timestamp        { return s.updatedAt }

func (s *Session) IsRevoked() bool {
	return s.revokedAt.IsValid()
}

func (s *Session) IsExpired(now fields.Timestamp) bool {
	return !s.expiresAt.Time().After(now.Time())
}

func (s *Session) IsValid(now fields.Timestamp) bool {
	return !s.IsRevoked() && !s.IsExpired(now)
}
