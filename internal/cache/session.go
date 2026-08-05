package cache

import (
	"context"
	"fmt"
	"net/netip"
	"time"

	"bonfire-api/internal/redis"
	"bonfire-api/internal/session"

	"github.com/google/uuid"
)

type SessionStore interface {
	Get(ctx context.Context, key string, dest interface{}) error
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key ...string) error
}

type Session struct {
	store SessionStore
}

func NewSession(store SessionStore) *Session {
	return &Session{store: store}
}

func sessionKey(id uuid.UUID) string {
	return fmt.Sprintf("session:%s", id.String())
}

type sessionDTO struct {
	ID               uuid.UUID  `json:"id"`
	UserID           uuid.UUID  `json:"user_id"`
	RefreshTokenHash []byte     `json:"refresh_token_hash"`
	ClientIP         string     `json:"client_ip"`
	UserAgent        string     `json:"user_agent"`
	OS               string     `json:"os"`
	Browser          string     `json:"browser"`
	ExpiresAt        time.Time  `json:"expires_at"`
	LastSeenAt       time.Time  `json:"last_seen_at"`
	RevokedAt        *time.Time `json:"revoked_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

func (s *Session) Get(ctx context.Context, id uuid.UUID) (*session.Session, error) {
	key := sessionKey(id)

	var dto sessionDTO
	err := s.store.Get(ctx, key, &dto)
	if redis.IsNotFoundError(err) {
		return nil, nil // Return nil on redis miss to let caller fall back to DB
	}
	if err != nil {
		return nil, redis.NewError(err, redis.ScopeSession)
	}

	clientIP, err := netip.ParseAddr(dto.ClientIP)
	if err != nil {
		return nil, redis.NewError(err, redis.ScopeSession)
	}

	tokenHash, err := session.NewRefreshTokenHash(dto.RefreshTokenHash)
	if err != nil {
		return nil, redis.NewError(err, redis.ScopeSession)
	}

	return session.Reconstitute(
		dto.ID,
		dto.UserID,
		tokenHash,
		clientIP,
		dto.UserAgent,
		dto.OS,
		dto.Browser,
		dto.ExpiresAt,
		dto.LastSeenAt,
		dto.CreatedAt,
		dto.UpdatedAt,
		dto.RevokedAt,
	), nil
}

func (s *Session) Set(ctx context.Context, sess *session.Session) error {
	ttl := time.Until(sess.ExpiresAt())
	if ttl <= 0 {
		return nil
	}

	dto := sessionDTO{
		ID:               sess.ID(),
		UserID:           sess.UserID(),
		RefreshTokenHash: sess.RefreshTokenHash().Bytes(),
		ClientIP:         sess.ClientIP().String(),
		UserAgent:        sess.UserAgent(),
		OS:               sess.OS(),
		Browser:          sess.Browser(),
		ExpiresAt:        sess.ExpiresAt(),
		LastSeenAt:       sess.LastSeenAt(),
		RevokedAt:        sess.RevokedAt(),
		CreatedAt:        sess.CreatedAt(),
		UpdatedAt:        sess.UpdatedAt(),
	}

	key := sessionKey(sess.ID())
	if err := s.store.Set(ctx, key, dto, ttl); err != nil {
		return redis.NewError(err, redis.ScopeSession)
	}

	return nil
}

func (s *Session) Delete(ctx context.Context, id uuid.UUID) error {
	key := sessionKey(id)

	if err := s.store.Delete(ctx, key); err != nil && !redis.IsNotFoundError(err) {
		return redis.NewError(err, redis.ScopeSession)
	}

	return nil
}
