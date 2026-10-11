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
	return &SessionRepository{store: store}
}

func (r *SessionRepository) Create(ctx context.Context, s *session.Session) (*session.Session, error) {
	if s == nil {
		return nil, ErrDomainPayloadRequired().Meta("resource", db.ResourceSession.String())
	}

	row, err := r.store.SessionCreate(ctx, db.SessionCreateParams{
		ID:               db.ToUUID(s.ID),
		UserID:           db.ToUUID(s.UserID),
		CreatedAt:        db.ToTimestamptz(s.CreatedAt),
		UpdatedAt:        db.ToTimestamptz(s.UpdatedAt),
		LastSeenAt:       db.ToTimestamptz(s.LastSeenAt),
		ExpiresAt:        db.ToTimestamptz(s.ExpiresAt),
		RevokedAt:        db.ToTimestamptzPtr(s.RevokedAt),
		ClientIP:         s.ClientIP,
		RefreshTokenHash: s.RefreshTokenHash[:],
		OS:               s.OS,
		Client:           s.Client,
		UserAgent:        s.UserAgent,
	})
	if err != nil {
		return nil, db.NewError(err, db.ResourceSession)
	}

	return sessionFromRow(row), nil
}

func (r *SessionRepository) Get(ctx context.Context, id uuid.UUID) (*session.Session, error) {
	row, err := r.store.SessionGet(ctx, db.ToUUID(id))
	if err != nil {
		return nil, db.NewError(err, db.ResourceSession)
	}

	return sessionFromRow(row), nil
}

func (r *SessionRepository) ListValidByUserID(ctx context.Context, userID uuid.UUID, now time.Time, limit int) ([]*session.Session, error) {
	rows, err := r.store.SessionListValidByUserID(ctx, db.SessionListValidByUserIDParams{
		UserID:   db.ToUUID(userID),
		Now:      db.ToTimestamptz(now),
		LimitVal: int32(limit),
	})
	if err != nil {
		return nil, db.NewError(err, db.ResourceSession)
	}

	return sessionsFromRows(rows), nil
}

func (r *SessionRepository) RotateRefreshTokenHash(
	ctx context.Context,
	id uuid.UUID,
	oldHash, newHash [32]byte,
	clientIP netip.Addr,
	userAgent string,
	os string,
	client string,
	expiresAt,
	now time.Time,
) (*session.Session, error) {
	row, err := r.store.SessionRotateRefreshTokenHash(ctx, db.SessionRotateRefreshTokenHashParams{
		ID:                  db.ToUUID(id),
		OldRefreshTokenHash: oldHash[:],
		NewRefreshTokenHash: newHash[:],
		ClientIP:            clientIP,
		UserAgent:           userAgent,
		OS:                  os,
		Client:              client,
		ExpiresAt:           db.ToTimestamptz(expiresAt),
		Now:                 db.ToTimestamptz(now),
	})
	if err != nil {
		return nil, db.NewError(err, db.ResourceSession)
	}

	return sessionFromRow(row), nil
}

func (r *SessionRepository) Revoke(ctx context.Context, id, userID uuid.UUID, now time.Time) error {
	err := r.store.SessionRevoke(ctx, db.SessionRevokeParams{
		ID:     db.ToUUID(id),
		UserID: db.ToUUID(userID),
		Now:    db.ToTimestamptz(now),
	})
	if err != nil {
		return db.NewError(err, db.ResourceSession)
	}

	return nil
}

func (r *SessionRepository) RevokeAll(ctx context.Context, userID uuid.UUID, now time.Time) ([]uuid.UUID, error) {
	ids, err := r.store.SessionRevokeAll(ctx, db.SessionRevokeAllParams{
		UserID: db.ToUUID(userID),
		Now:    db.ToTimestamptz(now),
	})
	if err != nil {
		return nil, db.NewError(err, db.ResourceSession)
	}

	return db.FromUUIDs(ids), nil
}

func (r *SessionRepository) DeleteBatchExpired(ctx context.Context, now time.Time, limitVal int) error {
	err := r.store.SessionDeleteBatchExpired(ctx, db.SessionDeleteBatchExpiredParams{
		Now:      db.ToTimestamptz(now),
		LimitVal: int32(limitVal),
	})
	if err != nil {
		return db.NewError(err, db.ResourceSession)
	}

	return nil
}

func (r *SessionRepository) DeleteBatchRevoked(ctx context.Context, now time.Time, limitVal int) error {
	err := r.store.SessionDeleteBatchRevoked(ctx, db.SessionDeleteBatchRevokedParams{
		Now:      db.ToTimestamptz(now),
		LimitVal: int32(limitVal),
	})
	if err != nil {
		return db.NewError(err, db.ResourceSession)
	}

	return nil
}

func sessionFromRow(row db.Session) *session.Session {
	var hash [32]byte
	copy(hash[:], row.RefreshTokenHash)

	return session.Reconstitute(
		db.FromUUID(row.ID),
		db.FromUUID(row.UserID),
		hash,
		row.ClientIP,
		row.UserAgent,
		row.OS,
		row.Client,
		db.FromTimestamptz(row.ExpiresAt),
		db.FromTimestamptz(row.LastSeenAt),
		db.FromTimestamptzPtr(row.RevokedAt),
		db.FromTimestamptz(row.CreatedAt),
		db.FromTimestamptz(row.UpdatedAt),
	)
}

func sessionsFromRows(rows []db.Session) []*session.Session {
	if len(rows) == 0 {
		return nil
	}

	sessions := make([]*session.Session, 0, len(rows))
	for _, row := range rows {
		sessions = append(sessions, sessionFromRow(row))
	}

	return sessions
}
