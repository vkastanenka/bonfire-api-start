// internal/repository/session.go
package repository

import (
	"context"
	"time"

	"bonfire-api/internal/apperr"
	"bonfire-api/internal/db"
	"bonfire-api/internal/session"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type Session struct {
	q db.Querier
}

func NewSession(q db.Querier) *Session {
	return &Session{q: q}
}

func (r *Session) Create(ctx context.Context, s *session.Session) error {
	_, err := r.q.SessionCreate(ctx, db.SessionCreateParams{
		ID:               pgtype.UUID{Bytes: s.ID(), Valid: s.ID() != uuid.Nil},
		UserID:           pgtype.UUID{Bytes: s.UserID(), Valid: true},
		RefreshTokenHash: s.RefreshTokenHash().Bytes(),
		ExpiresAt:        pgtype.Timestamptz{Time: s.ExpiresAt(), Valid: true},
		ClientIP:         s.ClientIP(),
		UserAgent:        s.UserAgent(),
		OS:               s.OS(),
		Browser:          s.Browser(),
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
	return sessionFromDB(row)
}

func (r *Session) Save(ctx context.Context, s *session.Session) error {
	if s.IsRevoked() {
		_, err := r.q.SessionUpdateRevoked(ctx, pgtype.UUID{Bytes: s.ID(), Valid: true})
		if err != nil {
			return db.NewError(err, db.EntitySession)
		}
		return nil
	}

	_, err := r.q.SessionUpdateRefreshToken(ctx, db.SessionUpdateRefreshTokenParams{
		ID:               pgtype.UUID{Bytes: s.ID(), Valid: true},
		RefreshTokenHash: s.RefreshTokenHash().Bytes(),
		ExpiresAt:        pgtype.Timestamptz{Time: s.ExpiresAt(), Valid: true},
	})
	if err != nil {
		return db.NewError(err, db.EntitySession)
	}

	return nil
}

func (r *Session) Delete(ctx context.Context, id uuid.UUID) error {
	err := r.q.SessionDelete(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return db.NewError(err, db.EntitySession)
	}
	return nil
}

func (r *Session) DeleteAllExcept(ctx context.Context, userID uuid.UUID, exceptSessionID uuid.UUID) error {
	err := r.q.SessionDeleteAllExcept(ctx, db.SessionDeleteAllExceptParams{
		UserID: pgtype.UUID{Bytes: userID, Valid: true},
		ID:     pgtype.UUID{Bytes: exceptSessionID, Valid: true},
	})
	if err != nil {
		return db.NewError(err, db.EntitySession)
	}
	return nil
}

// Reconstitution Helper
func sessionFromDB(row db.Session) (*session.Session, error) {
	tokenHash, err := session.NewRefreshTokenHash(row.RefreshTokenHash)
	if err != nil {
		return nil, apperr.NewInternal(err)
	}

	var revokedAt *time.Time
	if row.RevokedAt.Valid {
		revokedAt = &row.RevokedAt.Time
	}

	return session.Reconstitute(
		uuid.UUID(row.ID.Bytes),
		uuid.UUID(row.UserID.Bytes),
		tokenHash,
		row.ClientIP,
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

var _ session.Repository = (*Session)(nil)
