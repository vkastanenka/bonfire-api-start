package repository

import (
	"context"
	"fmt"
	"time"

	"bonfire-api/internal/db"
	"bonfire-api/internal/errs"
	"bonfire-api/internal/fields"
	"bonfire-api/internal/session"

	"github.com/google/uuid"
)

type SessionRepository struct {
	store *db.Store
}

func NewSessionRepository(store *db.Store) *SessionRepository {
	return &SessionRepository{
		store: store.WithEntity(db.EntitySession),
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

func (r *SessionRepository) SessionGet(ctx context.Context, id fields.ID) (*session.Session, error) {
	row, err := r.store.SessionGet(ctx, db.ToUUID(id.UUID()))
	if err != nil {
		return nil, r.store.Err(err)
	}

	return sessionFromRow(row)
}

func (r *SessionRepository) SessionListValidByUserID(ctx context.Context, userID fields.ID, now fields.Timestamp, limit int) ([]*session.Session, error) {
	rows, err := r.store.SessionListValidByUserID(ctx, db.SessionListValidByUserIDParams{
		UserID:   db.ToUUID(userID.UUID()),
		Now:      db.ToTimestamptz(now.Time()),
		LimitVal: int32(limit),
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

	return sessions, nil
}

func (r *SessionRepository) SessionRotateRefreshTokenHash(
	ctx context.Context,
	id fields.ID,
	oldHash, newHash fields.TokenHash,
	clientIP fields.IP,
	userAgent fields.UserAgent,
	expiresAt fields.Timestamp,
	now fields.Timestamp,
) (*session.Session, error) {
	row, err := r.store.SessionRotateRefreshTokenHash(ctx, db.SessionRotateRefreshTokenHashParams{
		ID:                  db.ToUUID(id.UUID()),
		OldRefreshTokenHash: oldHash.Bytes.Bytes(),
		NewRefreshTokenHash: newHash.Bytes.Bytes(),
		ClientIP:            clientIP.Addr(),
		UserAgent:           userAgent.String(),
		ExpiresAt:           db.ToTimestamptz(expiresAt.Time()),
		Now:                 db.ToTimestamptz(now.Time()),
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return sessionFromRow(row)
}

func (r *SessionRepository) SessionRevoke(ctx context.Context, id, userID fields.ID, now fields.Timestamp) error {
	err := r.store.SessionRevoke(ctx, db.SessionRevokeParams{
		ID:     db.ToUUID(id.UUID()),
		UserID: db.ToUUID(userID.UUID()),
		Now:    db.ToTimestamptz(now.Time()),
	})
	if err != nil {
		return r.store.Err(err)
	}

	return nil
}

func (r *SessionRepository) SessionRevokeAll(ctx context.Context, userID fields.ID, now fields.Timestamp) error {
	err := r.store.SessionRevokeAll(ctx, db.SessionRevokeAllParams{
		UserID: db.ToUUID(userID.UUID()),
		Now:    db.ToTimestamptz(now.Time()),
	})
	if err != nil {
		return r.store.Err(err)
	}

	return nil
}

func (r *SessionRepository) SessionDeleteBatchExpired(ctx context.Context, now time.Time, limitVal int) error {
	err := r.store.SessionDeleteBatchExpired(ctx, db.SessionDeleteBatchExpiredParams{
		Now:      db.ToTimestamptz(now),
		LimitVal: int32(limitVal),
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

	id, err := fields.ParseID(db.FromUUID[uuid.UUID](row.ID))
	if err != nil {
		return nil, mapErr("failed to parse session id from database", "id", sessionIDStr, err)
	}

	userIDVal := db.FromUUID[uuid.UUID](row.UserID)
	userID, err := fields.ParseID(db.FromUUID[uuid.UUID](row.UserID))
	if err != nil {
		return nil, mapErr("failed to parse user id from database", "user_id", userIDVal.String(), err)
	}

	tokenHash, err := fields.ParseTokenHash("refresh_token_hash", row.RefreshTokenHash)
	if err != nil {
		return nil, mapErr("failed to parse refresh token hash from database", "refresh_token_hash", row.RefreshTokenHash, err)
	}

	clientIP := fields.NewIP(row.ClientIP)
	if err != nil {
		return nil, mapErr("failed to parse client ip from database", "client_ip", row.ClientIP.String(), err)
	}

	userAgent, err := fields.ParseUserAgent("user_agent", row.UserAgent)
	if err != nil {
		return nil, mapErr("failed to parse user agent from database", "user_agent", row.UserAgent, err)
	}

	os, err := fields.ParseOS("os", row.OS)
	if err != nil {
		return nil, mapErr("failed to parse os from database", "os", row.OS, err)
	}

	client, err := fields.ParseClient("client", row.Client)
	if err != nil {
		return nil, mapErr("failed to parse client from database", "client", row.Client, err)
	}

	expiresAt := fields.NewTimestamp(db.FromTimestamptz(row.ExpiresAt))
	lastSeenAt := fields.NewTimestamp(db.FromTimestamptz(row.LastSeenAt))
	revokedAt := fields.NewTimestamp(db.FromTimestamptz(row.RevokedAt))
	createdAt := fields.NewTimestamp(db.FromTimestamptz(row.CreatedAt))
	updatedAt := fields.NewTimestamp(db.FromTimestamptz(row.UpdatedAt))

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
