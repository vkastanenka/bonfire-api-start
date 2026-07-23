package repository

import (
	"context"
	"net/netip"

	"bonfire-api/internal/db"
	"bonfire-api/internal/errs"
	"bonfire-api/internal/session"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type SessionStore interface {
	SessionCreate(ctx context.Context, arg db.SessionCreateParams) (db.Session, error)
	SessionSave(ctx context.Context, arg db.SessionSaveParams) (db.Session, error)
	SessionGet(ctx context.Context, id pgtype.UUID) (db.Session, error)
	SessionDelete(ctx context.Context, id pgtype.UUID) error
	SessionDeleteAllByUserID(ctx context.Context, userID pgtype.UUID) error
	SessionDeleteAllExcept(ctx context.Context, arg db.SessionDeleteAllExceptParams) error
	SessionDeleteAllExpired(ctx context.Context) error
}

type Session struct {
	store SessionStore
}

func NewSession(store SessionStore) *Session {
	return &Session{store: store}
}

func (r *Session) Create(ctx context.Context, s *session.Session) error {
	_, err := r.store.SessionCreate(ctx, db.SessionCreateParams{
		ID:               db.UUID(s.ID()),
		UserID:           db.UUID(s.UserID()),
		RefreshTokenHash: s.RefreshTokenHash().Bytes(),
		ExpiresAt:        db.Timestamptz(s.ExpiresAt()),
		ClientIP:         s.ClientIP(),
		UserAgent:        s.UserAgent(),
		OS:               s.OS(),
		Browser:          s.Browser(),
		LastSeenAt:       db.Timestamptz(s.LastSeenAt()),
		RevokedAt:        db.TimestamptzPtr(s.RevokedAt()),
		CreatedAt:        db.Timestamptz(s.CreatedAt()),
		UpdatedAt:        db.Timestamptz(s.UpdatedAt()),
	})
	if err != nil {
		return db.NewError(err, db.EntitySession)
	}

	return nil
}

func (r *Session) Save(ctx context.Context, s *session.Session) error {
	_, err := r.store.SessionSave(ctx, db.SessionSaveParams{
		ID:               db.UUID(s.ID()),
		RefreshTokenHash: s.RefreshTokenHash().Bytes(),
		ExpiresAt:        db.Timestamptz(s.ExpiresAt()),
		LastSeenAt:       db.Timestamptz(s.LastSeenAt()),
		RevokedAt:        db.TimestamptzPtr(s.RevokedAt()),
		UpdatedAt:        db.Timestamptz(s.UpdatedAt()),
	})
	if err != nil {
		return db.NewError(err, db.EntitySession)
	}

	return nil
}

func (r *Session) Get(ctx context.Context, id uuid.UUID) (*session.Session, error) {
	row, err := r.store.SessionGet(ctx, db.UUID(id))
	if err != nil {
		return nil, db.NewError(err, db.EntitySession)
	}

	return sessionFromRow(row)
}

func (r *Session) Delete(ctx context.Context, id uuid.UUID) error {
	err := r.store.SessionDelete(ctx, db.UUID(id))
	if err != nil {
		return db.NewError(err, db.EntitySession)
	}

	return nil
}

func (r *Session) DeleteAllByUserID(ctx context.Context, userID uuid.UUID) error {
	err := r.store.SessionDeleteAllByUserID(ctx, db.UUID(userID))
	if err != nil {
		return db.NewError(err, db.EntitySession)
	}

	return nil
}

func (r *Session) DeleteAllExcept(ctx context.Context, userID, currentSessionID uuid.UUID) error {
	err := r.store.SessionDeleteAllExcept(ctx, db.SessionDeleteAllExceptParams{
		UserID: db.UUID(userID),
		ID:     db.UUID(currentSessionID),
	})
	if err != nil {
		return db.NewError(err, db.EntitySession)
	}

	return nil
}

func (r *Session) DeleteAllExpired(ctx context.Context) error {
	err := r.store.SessionDeleteAllExpired(ctx)
	if err != nil {
		return db.NewError(err, db.EntitySession)
	}

	return nil
}

func sessionFromRow(row db.Session) (*session.Session, error) {
	sessionID := uuid.UUID(row.ID.Bytes).String()

	clientIP, err := netip.ParseAddr(row.ClientIP.String())
	if err != nil {
		return nil, errs.Internal("failed to parse client IP from database").
			Wrap(err).
			Reason("CORRUPT_DATABASE_RECORD").
			Meta("client_ip", row.ClientIP.String()).
			Resource("Session", sessionID, "", "database row mapping")
	}

	tokenHash, err := session.NewRefreshTokenHash(row.RefreshTokenHash)
	if err != nil {
		return nil, errs.Internal("failed to parse refresh token hash from database").
			Wrap(err).
			Reason("CORRUPT_DATABASE_RECORD").
			Resource("Session", sessionID, "", "database row mapping")
	}

	return session.Reconstitute(
		uuid.UUID(row.ID.Bytes),
		uuid.UUID(row.UserID.Bytes),
		tokenHash,
		clientIP,
		row.UserAgent,
		row.OS,
		row.Browser,
		row.ExpiresAt.Time.UTC(),
		row.LastSeenAt.Time.UTC(),
		row.CreatedAt.Time.UTC(),
		row.UpdatedAt.Time.UTC(),
		db.TimePtr(row.RevokedAt),
	), nil
}
