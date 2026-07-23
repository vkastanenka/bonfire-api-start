package repository

import (
	"context"
	"net/netip"
	"time"

	"bonfire-api/internal/db"
	"bonfire-api/internal/errs"
	"bonfire-api/internal/session"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type Session struct {
	q db.Store
}

var _ session.Repository = (*Session)(nil)

func NewSession(q db.Store) *Session {
	return &Session{q: q}
}

func (r *Session) Create(ctx context.Context, s *session.Session) error {
	_, err := r.q.SessionCreate(ctx, db.SessionCreateParams{
		ID:               pgtype.UUID{Bytes: s.ID(), Valid: true},
		UserID:           pgtype.UUID{Bytes: s.UserID(), Valid: true},
		RefreshTokenHash: s.RefreshTokenHash().Bytes(),
		ExpiresAt:        pgtype.Timestamptz{Time: s.ExpiresAt(), Valid: true},
		ClientIP:         s.ClientIP(),
		UserAgent:        s.UserAgent(),
		OS:               s.OS(),
		Browser:          s.Browser(),
		LastSeenAt:       pgtype.Timestamptz{Time: s.LastSeenAt(), Valid: true},
		RevokedAt:        timeToTimestamptz(s.RevokedAt()),
		CreatedAt:        pgtype.Timestamptz{Time: s.CreatedAt(), Valid: true},
		UpdatedAt:        pgtype.Timestamptz{Time: s.UpdatedAt(), Valid: true},
	})
	if err != nil {
		return db.NewError(err, db.EntitySession)
	}

	return nil
}

func (r *Session) Save(ctx context.Context, s *session.Session) error {
	_, err := r.q.SessionSave(ctx, db.SessionSaveParams{
		ID:               pgtype.UUID{Bytes: s.ID(), Valid: true},
		RefreshTokenHash: s.RefreshTokenHash().Bytes(),
		ExpiresAt:        pgtype.Timestamptz{Time: s.ExpiresAt(), Valid: true},
		LastSeenAt:       pgtype.Timestamptz{Time: s.LastSeenAt(), Valid: true},
		RevokedAt:        timeToTimestamptz(s.RevokedAt()),
		UpdatedAt:        pgtype.Timestamptz{Time: s.UpdatedAt(), Valid: true},
	})
	if err != nil {
		return db.NewError(err, db.EntitySession)
	}

	return nil
}

func (r *Session) Get(ctx context.Context, id uuid.UUID) (*session.Session, error) {
	row, err := r.q.SessionGet(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return nil, db.NewError(err, db.EntitySession)
	}

	return sessionFromRow(row)
}

func (r *Session) Delete(ctx context.Context, id uuid.UUID) error {
	err := r.q.SessionDelete(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return db.NewError(err, db.EntitySession)
	}

	return nil
}

func (r *Session) DeleteAllByUserID(ctx context.Context, userID uuid.UUID) error {
	err := r.q.SessionDeleteAllByUserID(ctx, pgtype.UUID{Bytes: userID, Valid: true})
	if err != nil {
		return db.NewError(err, db.EntitySession)
	}

	return nil
}

func (r *Session) DeleteAllExcept(ctx context.Context, userID, currentSessionID uuid.UUID) error {
	err := r.q.SessionDeleteAllExcept(ctx, db.SessionDeleteAllExceptParams{
		UserID: pgtype.UUID{Bytes: userID, Valid: true},
		ID:     pgtype.UUID{Bytes: currentSessionID, Valid: true},
	})
	if err != nil {
		return db.NewError(err, db.EntitySession)
	}

	return nil
}

func (r *Session) DeleteAllExpired(ctx context.Context) error {
	err := r.q.SessionDeleteAllExpired(ctx)
	if err != nil {
		return db.NewError(err, db.EntitySession)
	}

	return nil
}

func sessionFromRow(row db.Session) (*session.Session, error) {
	clientIP, err := netip.ParseAddr(row.ClientIP.String())
	if err != nil {
		return nil, errs.Internal("").Wrap(err)
	}

	var revokedAt *time.Time
	if row.RevokedAt.Valid {
		revokedAt = &row.RevokedAt.Time
	}

	tokenHash, err := session.NewRefreshTokenHash(row.RefreshTokenHash)
	if err != nil {
		return nil, errs.Internal("").Wrap(err)
	}

	return session.Reconstitute(
		uuid.UUID(row.ID.Bytes),
		uuid.UUID(row.UserID.Bytes),
		tokenHash,
		clientIP,
		row.UserAgent,
		row.OS,
		row.Browser,
		row.ExpiresAt.Time,
		row.LastSeenAt.Time,
		row.CreatedAt.Time,
		row.UpdatedAt.Time,
		revokedAt,
	), nil
}
