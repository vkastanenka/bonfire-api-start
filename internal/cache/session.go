package cache

import (
	"context"
	"fmt"
	"time"

	"bonfire-api/internal/fields"
	"bonfire-api/internal/redis"
	"bonfire-api/internal/session"

	"github.com/google/uuid"
)

func SessionKey(id fields.ID) string {
	return fmt.Sprintf("sessions:%s", id.String())
}

type Session struct {
	ID               uuid.UUID `json:"id"`
	UserID           uuid.UUID `json:"user_id"`
	RefreshTokenHash []byte    `json:"refresh_token_hash"`
	ClientIP         string    `json:"client_ip"`
	UserAgent        string    `json:"user_agent"`
	OS               string    `json:"os"`
	Client           string    `json:"client"`
	ExpiresAt        int64     `json:"expires_at"`
	LastSeenAt       int64     `json:"last_seen_at"`
	RevokedAt        int64     `json:"revoked_at"`
	CreatedAt        int64     `json:"created_at"`
	UpdatedAt        int64     `json:"updated_at"`
}

func (s Session) ToDomain() (*session.Session, error) {
	id, err := fields.ParseRequiredID("id", s.ID)
	if err != nil {
		return nil, err
	}
	userID, err := fields.ParseRequiredID("user_id", s.UserID)
	if err != nil {
		return nil, err
	}
	hash, err := session.ParseRefreshTokenHash("refresh_token_hash", s.RefreshTokenHash)
	if err != nil {
		return nil, err
	}
	clientIP, err := session.ParseClientIP("client_ip", s.ClientIP)
	if err != nil {
		return nil, err
	}
	userAgent, err := session.ParseUserAgent("user_agent", s.UserAgent)
	if err != nil {
		return nil, err
	}
	os, err := session.ParseOS("os", s.OS)
	if err != nil {
		return nil, err
	}
	client, err := session.ParseClient("client", s.Client)
	if err != nil {
		return nil, err
	}
	expiresAt, err := session.ParseExpiresAt("expires_at", time.Unix(s.ExpiresAt, 0), time.Time{})
	if err != nil {
		return nil, err
	}

	return session.Reconstitute(
		id,
		userID,
		hash,
		clientIP,
		userAgent,
		os,
		client,
		expiresAt,
		fields.NewTimestampFromUnix(s.LastSeenAt),
		fields.NewTimestampFromUnix(s.RevokedAt),
		fields.NewTimestampFromUnix(s.CreatedAt),
		fields.NewTimestampFromUnix(s.UpdatedAt),
	), nil
}

func SessionFromDomain(s *session.Session) *Session {
	return &Session{
		ID:               s.ID().UUID(),
		UserID:           s.UserID().UUID(),
		RefreshTokenHash: s.RefreshTokenHash().Bytes.Bytes(),
		ClientIP:         s.ClientIP().String(),
		UserAgent:        s.UserAgent().String(),
		OS:               s.OS().String(),
		Client:           s.Client().String(),
		ExpiresAt:        s.ExpiresAt().Unix(),
		LastSeenAt:       s.LastSeenAt().Unix(),
		RevokedAt:        s.RevokedAt().Unix(),
		CreatedAt:        s.CreatedAt().Unix(),
		UpdatedAt:        s.UpdatedAt().Unix(),
	}
}

type SessionCache struct {
	engine *KeyCache[fields.ID, Session]
}

func NewSessionCache(store *redis.Store, ttl time.Duration) *SessionCache {
	engine := NewKeyCache[fields.ID, Session](
		store.WithScope(redis.ScopeSession),
		ttl,
		SessionKey,
	)
	return &SessionCache{engine: engine}
}

func (s *SessionCache) Set(ctx context.Context, sess *session.Session) error {
	if sess == nil {
		return nil
	}
	return s.engine.Set(ctx, sess.ID(), SessionFromDomain(sess))
}

func (s *SessionCache) SetBatch(ctx context.Context, sessions []*session.Session) error {
	if len(sessions) == 0 {
		return nil
	}
	items := make(map[fields.ID]*Session, len(sessions))
	for _, sess := range sessions {
		if sess != nil {
			items[sess.ID()] = SessionFromDomain(sess)
		}
	}
	return s.engine.SetBatch(ctx, items)
}

func (s *SessionCache) Get(ctx context.Context, sessionID fields.ID) (*session.Session, error) {
	if !sessionID.IsValid() {
		return nil, nil
	}
	cached, err := s.engine.Get(ctx, sessionID)
	if err != nil || cached == nil {
		return nil, err
	}
	return cached.ToDomain()
}

func (s *SessionCache) GetBatch(ctx context.Context, sessionIDs []fields.ID) (map[fields.ID]*session.Session, []fields.ID, error) {
	found, missing, err := s.engine.GetBatch(ctx, sessionIDs)
	if err != nil {
		return nil, nil, err
	}

	result := make(map[fields.ID]*session.Session, len(found))
	for id, cached := range found {
		domainSession, err := cached.ToDomain()
		if err != nil {
			missing = append(missing, id)
			continue
		}
		result[id] = domainSession
	}

	return result, missing, nil
}

func (s *SessionCache) Delete(ctx context.Context, sessionID fields.ID) error {
	if !sessionID.IsValid() {
		return nil
	}
	return s.engine.Delete(ctx, sessionID)
}

func (s *SessionCache) DeleteBatch(ctx context.Context, sessionIDs []fields.ID) error {
	return s.engine.DeleteBatch(ctx, sessionIDs)
}
