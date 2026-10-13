package auth

import (
	"bonfire-api/internal/redis"
	"context"
	"encoding/json"
	"net/netip"
	"time"

	"github.com/google/uuid"
	redisdriver "github.com/redis/go-redis/v9"
)

const (
	SessionCacheTTL    = 24 * time.Hour
	sessionCacheMinTTL = 1 * time.Second
)

func SessionCacheKey(id uuid.UUID) string { return "session:" + "{" + id.String() + "}" }

type SessionCache struct {
	client redisdriver.Cmdable
}

func NewSessionCache(client redisdriver.Cmdable) *SessionCache {
	return &SessionCache{client: client}
}

func (c *SessionCache) Get(ctx context.Context, id uuid.UUID) (*Session, error) {
	return redis.GetAndUnmarshal(
		ctx,
		c.client,
		redis.ResourceSession,
		SessionCacheKey(id),
		UnmarshalSessionCacheDTO,
	)
}

func (c *SessionCache) Set(ctx context.Context, sess *Session) error {
	if sess == nil {
		return nil
	}

	ttl, ok := CalcSessionCacheTTL(sess)
	if !ok {
		return c.Delete(ctx, sess.ID)
	}

	return redis.MarshalAndSet(
		ctx,
		c.client,
		redis.ResourceSession,
		SessionCacheKey(sess.ID),
		sess,
		ttl,
		MarshalSessionCacheDTO,
	)
}

func (c *SessionCache) Delete(ctx context.Context, id uuid.UUID) error {
	if err := c.client.Del(ctx, SessionCacheKey(id)).Err(); err != nil {
		return redis.NewError(err, redis.ResourceSession)
	}
	return nil
}

func (c *SessionCache) DeleteBatch(ctx context.Context, ids []uuid.UUID) error {
	return redis.DeleteBatch(ctx, c.client, redis.ResourceSession, SessionCacheKey, ids)
}

func CalcSessionCacheTTL(sess *Session) (time.Duration, bool) {
	if sess.ExpiresAt.IsZero() {
		return SessionCacheTTL, true
	}

	remaining := time.Until(sess.ExpiresAt)
	if remaining < sessionCacheMinTTL {
		return 0, false
	}

	return remaining, true
}

type SessionCacheDTO struct {
	ID               uuid.UUID  `json:"id"`
	UserID           uuid.UUID  `json:"user_id"`
	RefreshTokenHash string     `json:"refresh_token_hash"`
	ClientIP         netip.Addr `json:"client_ip"`
	UserAgent        string     `json:"user_agent"`
	OS               string     `json:"os"`
	Client           string     `json:"client"`
	ExpiresAt        time.Time  `json:"expires_at"`
	LastSeenAt       time.Time  `json:"last_seen_at"`
	RevokedAt        *time.Time `json:"revoked_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

func NewSessionCacheDTO(s *Session) SessionCacheDTO {
	return SessionCacheDTO{
		ID:               s.ID,
		UserID:           s.UserID,
		RefreshTokenHash: string(s.RefreshTokenHash[:]),
		ClientIP:         s.ClientIP,
		UserAgent:        s.UserAgent,
		OS:               s.OS,
		Client:           s.Client,
		ExpiresAt:        s.ExpiresAt,
		LastSeenAt:       s.LastSeenAt,
		RevokedAt:        s.RevokedAt,
		CreatedAt:        s.CreatedAt,
		UpdatedAt:        s.UpdatedAt,
	}
}

func (dto SessionCacheDTO) ToDomain() *Session {
	var hash [32]byte
	copy(hash[:], dto.RefreshTokenHash)

	return Reconstitute(
		dto.ID,
		dto.UserID,
		hash,
		dto.ClientIP,
		dto.UserAgent,
		dto.OS,
		dto.Client,
		dto.ExpiresAt,
		dto.LastSeenAt,
		dto.RevokedAt,
		dto.CreatedAt,
		dto.UpdatedAt,
	)
}

func UnmarshalSessionCacheDTO(b []byte) (*Session, error) {
	var dto SessionCacheDTO
	if err := json.Unmarshal(b, &dto); err != nil {
		return nil, err
	}
	return dto.ToDomain(), nil
}

func MarshalSessionCacheDTO(sess *Session) ([]byte, error) {
	dto := NewSessionCacheDTO(sess)
	bytes, err := json.Marshal(dto)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}
