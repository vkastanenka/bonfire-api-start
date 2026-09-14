package repository

import (
	"context"
	"net/netip"
	"time"

	"bonfire-api/internal/db"
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

func (r *SessionRepository) Create(ctx context.Context, s *session.Session) (*session.Session, error) {
	row, err := r.store.SessionCreate(ctx, db.SessionCreateParams{
		ID:               db.ToUUID(s.ID),
		UserID:           db.ToUUID(s.UserID),
		CreatedAt:        db.ToTimestamptz(s.CreatedAt),
		UpdatedAt:        db.ToTimestamptz(s.UpdatedAt),
		LastSeenAt:       db.ToTimestamptz(s.LastSeenAt),
		ExpiresAt:        db.ToTimestamptz(s.ExpiresAt),
		RevokedAt:        db.ToTimestamptzPtr(s.RevokedAt),
		ClientIP:         s.ClientIP,
		RefreshTokenHash: []byte(s.RefreshTokenHash),
		OS:               s.OS,
		Client:           s.Client,
		UserAgent:        s.UserAgent,
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return sessionFromRow(row)
}

func (r *SessionRepository) Get(ctx context.Context, id uuid.UUID) (*session.Session, error) {
	row, err := r.store.SessionGet(ctx, db.ToUUID(id))
	if err != nil {
		return nil, r.store.Err(err)
	}

	return sessionFromRow(row)
}

func (r *SessionRepository) ListValidByUserID(ctx context.Context, userID uuid.UUID, now time.Time, limit int) ([]*session.Session, error) {
	rows, err := r.store.SessionListValidByUserID(ctx, db.SessionListValidByUserIDParams{
		UserID:   db.ToUUID(userID),
		Now:      db.ToTimestamptz(now),
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

func (r *SessionRepository) RotateRefreshTokenHash(
	ctx context.Context,
	id uuid.UUID,
	oldHash, newHash string,
	clientIP netip.Addr,
	userAgent string,
	expiresAt,
	now time.Time,
) (*session.Session, error) {
	row, err := r.store.SessionRotateRefreshTokenHash(ctx, db.SessionRotateRefreshTokenHashParams{
		ID:                  db.ToUUID(id),
		OldRefreshTokenHash: []byte(oldHash),
		NewRefreshTokenHash: []byte(newHash),
		ClientIP:            clientIP,
		UserAgent:           userAgent,
		ExpiresAt:           db.ToTimestamptz(expiresAt),
		Now:                 db.ToTimestamptz(now),
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return sessionFromRow(row)
}

func (r *SessionRepository) Revoke(ctx context.Context, id, userID uuid.UUID, now time.Time) error {
	err := r.store.SessionRevoke(ctx, db.SessionRevokeParams{
		ID:     db.ToUUID(id),
		UserID: db.ToUUID(userID),
		Now:    db.ToTimestamptz(now),
	})
	if err != nil {
		return r.store.Err(err)
	}

	return nil
}

func (r *SessionRepository) RevokeAll(ctx context.Context, userID uuid.UUID, now time.Time) ([]uuid.UUID, error) {
	dbIDs, err := r.store.SessionRevokeAll(ctx, db.SessionRevokeAllParams{
		UserID: db.ToUUID(userID),
		Now:    db.ToTimestamptz(now),
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	ids := make([]uuid.UUID, len(dbIDs))
	for i, dbID := range dbIDs {
		ids[i] = uuid.UUID(db.FromUUID[uuid.UUID](dbID))
	}

	return ids, nil
}

func (r *SessionRepository) DeleteBatchExpired(ctx context.Context, now time.Time, limitVal int) error {
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
	return session.Reconstitute(
		db.FromUUID[uuid.UUID](row.ID),
		db.FromUUID[uuid.UUID](row.UserID),
		string(row.RefreshTokenHash),
		row.ClientIP,
		row.UserAgent,
		row.OS,
		row.Client,
		db.FromTimestamptz(row.ExpiresAt),
		db.FromTimestamptz(row.LastSeenAt),
		db.FromTimestamptzPtr(row.RevokedAt),
		db.FromTimestamptz(row.CreatedAt),
		db.FromTimestamptz(row.UpdatedAt),
	), nil
}
