package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"bonfire-api/internal/cache"
	"bonfire-api/internal/db"
	"bonfire-api/internal/errs"
	"bonfire-api/internal/fields"
	"bonfire-api/internal/redis"
	"bonfire-api/internal/session"

	"github.com/google/uuid"
)

type SessionRepository struct {
	store *db.Store
	cache *cache.SessionCache
}

func NewSessionRepository(store *db.Store, cache *cache.SessionCache) *SessionRepository {
	return &SessionRepository{
		store: store.WithEntity(db.EntitySession),
		cache: cache,
	}
}

func (r *SessionRepository) SessionCreate(ctx context.Context, s *session.Session) (*session.Session, error) {
	row, err := r.store.SessionCreate(ctx, db.SessionCreateParams{
		ID:               db.ToUUID(s.ID().UUID()),
		UserID:           db.ToUUID(s.UserID().UUID()),
		CreatedAt:        db.ToTimestamptz(s.CreatedAt().Time()),
		UpdatedAt:        db.ToTimestamptz(s.UpdatedAt().Time()),
		LastSeenAt:       db.ToTimestamptz(s.LastSeenAt().Time()),
		ExpiresAt:        db.ToTimestamptz(s.ExpiresAt().Time()),
		RevokedAt:        db.ToTimestamptzPtr(s.RevokedAt().TimePtr()),
		ClientIP:         s.ClientIP().Addr(),
		RefreshTokenHash: s.RefreshTokenHash().Bytes.Bytes(),
		OS:               s.OS().String(),
		Client:           s.Client().String(),
		UserAgent:        s.UserAgent().String(),
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return sessionFromRow(row)
}

func (r *SessionRepository) SessionDeleteExpiredBatch(ctx context.Context, now time.Time, batchLimit int32) error {
	err := r.store.SessionDeleteExpiredBatch(ctx, db.SessionDeleteExpiredBatchParams{
		Now:        db.ToTimestamptz(now),
		BatchLimit: batchLimit,
	})
	if err != nil {
		return r.store.Err(err)
	}

	return nil
}

func (r *SessionRepository) SessionGet(ctx context.Context, id fields.ID) (*session.Session, error) {
	if r.cache != nil {
		s, err := r.cache.Get(ctx, id)
		if err == nil && s != nil {
			return s, nil
		}

		if err != nil && !errors.Is(err, redis.ErrCacheMiss) {
			slog.WarnContext(ctx, "session cache read failed, falling back to database",
				"session_id", id.String(),
				"error", err,
				"scope", redis.ScopeSession,
			)
		}
	}

	row, err := r.store.SessionGet(ctx, db.ToUUID(id.UUID()))
	if err != nil {
		return nil, r.store.Err(err)
	}

	sess, err := sessionFromRow(row)
	if err != nil {
		return nil, err
	}

	if r.cache != nil {
		if cacheErr := r.cache.Set(ctx, sess); cacheErr != nil {
			slog.WarnContext(ctx, "failed to backfill session cache",
				"session_id", sess.ID().String(),
				"error", cacheErr,
				"scope", redis.ScopeSession,
			)
		}
	}

	return sess, nil
}

func (r *SessionRepository) SessionRotateRefreshToken(ctx context.Context, id fields.ID, oldHash, newHash session.RefreshTokenHash, expiresAt, lastSeenAt, updatedAt fields.Timestamp, clientIP session.ClientIP, userAgent session.UserAgent, now fields.Timestamp) (*session.Session, error) {
	row, err := r.store.SessionRotateRefreshToken(ctx, db.SessionRotateRefreshTokenParams{
		ID:                  db.ToUUID(id.UUID()),
		OldRefreshTokenHash: oldHash.Bytes.Bytes(),
		NewRefreshTokenHash: newHash.Bytes.Bytes(),
		ExpiresAt:           db.ToTimestamptz(expiresAt.Time()),
		LastSeenAt:          db.ToTimestamptz(lastSeenAt.Time()),
		UpdatedAt:           db.ToTimestamptz(updatedAt.Time()),
		ClientIP:            clientIP.Addr(),
		UserAgent:           userAgent.String(),
		Now:                 db.ToTimestamptz(now.Time()),
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return sessionFromRow(row)
}

func (r *SessionRepository) SessionUserGetBatch(ctx context.Context, userID fields.ID, now fields.Timestamp, limit int32) ([]*session.Session, error) {
	rows, err := r.store.SessionUserGetBatch(ctx, db.SessionUserGetBatchParams{
		UserID:   db.ToUUID(userID.UUID()),
		Now:      db.ToTimestamptz(now.Time()),
		LimitVal: limit,
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	sessions := make([]*session.Session, 0, len(rows))
	for _, row := range rows {
		s, err := sessionFromRow(row)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}

	if r.cache != nil && len(sessions) > 0 {
		if cacheErr := r.cache.SetBatch(ctx, sessions); cacheErr != nil {
			slog.WarnContext(ctx, "failed to backfill session cache batch",
				"user_id", userID.String(),
				"error", cacheErr,
				"scope", redis.ScopeSession,
			)
		}
	}

	return sessions, nil
}

func (r *SessionRepository) SessionUserRevoke(ctx context.Context, id, userID fields.ID, revokedAt, updatedAt fields.Timestamp) error {
	err := r.store.SessionUserRevoke(ctx, db.SessionUserRevokeParams{
		ID:        db.ToUUID(id.UUID()),
		UserID:    db.ToUUID(userID.UUID()),
		RevokedAt: db.ToTimestamptz(revokedAt.Time()),
		UpdatedAt: db.ToTimestamptz(updatedAt.Time()),
	})
	if err != nil {
		return r.store.Err(err)
	}

	return nil
}

func (r *SessionRepository) SessionUserRevokeAll(ctx context.Context, userID fields.ID, revokedAt, updatedAt fields.Timestamp) error {
	err := r.store.SessionUserRevokeAll(ctx, db.SessionUserRevokeAllParams{
		UserID:    db.ToUUID(userID.UUID()),
		RevokedAt: db.ToTimestamptz(revokedAt.Time()),
		UpdatedAt: db.ToTimestamptz(updatedAt.Time()),
	})
	if err != nil {
		return r.store.Err(err)
	}

	return nil
}

func sessionFromRow(row db.Session) (*session.Session, error) {
	sessionID := db.FromUUID[uuid.UUID](row.ID)
	sessionIDStr := sessionID.String()

	mapErr := func(msg, key string, val any, err error) *errs.Error {
		return errs.Internal(msg).
			Wrap(err).
			Reason("CORRUPT_DATABASE_RECORD").
			Meta(key, fmt.Sprintf("%v", val)).
			Resource("Session", sessionIDStr, "", "database row mapping")
	}

	id, err := fields.ParseRequiredID("id", sessionID)
	if err != nil {
		return nil, mapErr("failed to parse session id from database", "id", sessionIDStr, err)
	}

	userIDVal := db.FromUUID[uuid.UUID](row.UserID)
	userID, err := fields.ParseRequiredID("user_id", userIDVal)
	if err != nil {
		return nil, mapErr("failed to parse user id from database", "user_id", userIDVal.String(), err)
	}

	tokenHash, err := session.ParseRefreshTokenHash("refresh_token_hash", row.RefreshTokenHash)
	if err != nil {
		return nil, mapErr("failed to parse refresh token hash from database", "refresh_token_hash", row.RefreshTokenHash, err)
	}

	ipStr := row.ClientIP.String()
	clientIP, err := session.ParseClientIP("client_ip", ipStr)
	if err != nil {
		return nil, mapErr("failed to parse client ip from database", "client_ip", ipStr, err)
	}

	userAgent, err := session.ParseUserAgent("user_agent", row.UserAgent)
	if err != nil {
		return nil, mapErr("failed to parse user agent from database", "user_agent", row.UserAgent, err)
	}

	os, err := session.ParseOS("os", row.OS)
	if err != nil {
		return nil, mapErr("failed to parse os from database", "os", row.OS, err)
	}

	client, err := session.ParseClient("client", row.Client)
	if err != nil {
		return nil, mapErr("failed to parse client from database", "client", row.Client, err)
	}

	expiresAt, err := session.ParseExpiresAt("expires_at", row.ExpiresAt.Time.UTC(), time.Time{})
	if err != nil {
		return nil, mapErr("failed to parse expires_at from database", "expires_at", row.ExpiresAt.Time.UTC(), err)
	}

	lastSeenAt := fields.NewTimestampFromTime(db.FromTimestamptz(row.LastSeenAt))
	revokedAt := fields.NewTimestampFromTime(db.FromTimestamptz(row.RevokedAt))
	createdAt := fields.NewTimestampFromTime(db.FromTimestamptz(row.CreatedAt))
	updatedAt := fields.NewTimestampFromTime(db.FromTimestamptz(row.UpdatedAt))

	return session.Reconstitute(
		id,
		userID,
		tokenHash,
		clientIP,
		userAgent,
		os,
		client,
		expiresAt,
		lastSeenAt,
		revokedAt,
		createdAt,
		updatedAt,
	), nil
}
