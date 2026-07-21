package repository

import (
	"context"

	"bonfire-api/internal/db"
	"bonfire-api/internal/pkg/ptr"
	"bonfire-api/internal/session"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type Session struct {
	store db.Store
}

func NewSession(store db.Store) *Session {
	return &Session{store: store}
}

func (r *Session) Create(ctx context.Context, p session.CreateParams) (session.Session, error) {
	row, err := r.store.SessionCreate(ctx, db.SessionCreateParams{
		ID:               pgtype.UUID{Bytes: p.ID, Valid: p.ID != uuid.Nil},
		UserID:           pgtype.UUID{Bytes: p.UserID, Valid: true},
		RefreshTokenHash: p.RefreshTokenHash,
		ExpiresAt:        pgtype.Timestamptz{Time: p.ExpiresAt, Valid: true},
		ClientIP:         p.ClientIP,
		UserAgent:        p.UserAgent,
		OS:               p.OS,
		Browser:          p.Browser,
	})
	if err != nil {
		return session.Session{}, NewError(err, EntitySession)
	}

	return sessionFromDB(row), nil
}

func (r *Session) Get(ctx context.Context, id uuid.UUID) (session.Session, error) {
	row, err := r.store.SessionGet(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return session.Session{}, NewError(err, EntitySession)
	}
	return sessionFromDB(row), nil
}

func (r *Session) UpdateRefreshToken(ctx context.Context, p session.UpdateRefreshTokenParams) (session.Session, error) {
	row, err := r.store.SessionUpdateRefreshToken(ctx, db.SessionUpdateRefreshTokenParams{
		ID:               pgtype.UUID{Bytes: p.ID, Valid: true},
		RefreshTokenHash: p.RefreshTokenHash,
		ExpiresAt:        pgtype.Timestamptz{Time: p.ExpiresAt, Valid: true},
	})
	if err != nil {
		return session.Session{}, NewError(err, EntitySession)
	}
	return sessionFromDB(row), nil
}

func (r *Session) UpdateLastSeen(ctx context.Context, id uuid.UUID) (session.Session, error) {
	row, err := r.store.SessionUpdateLastSeen(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return session.Session{}, NewError(err, EntitySession)
	}
	return sessionFromDB(row), nil
}

func (r *Session) Revoke(ctx context.Context, id uuid.UUID) (session.Session, error) {
	row, err := r.store.SessionUpdateRevoked(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return session.Session{}, NewError(err, EntitySession)
	}
	return sessionFromDB(row), nil
}

func (r *Session) Delete(ctx context.Context, id uuid.UUID) error {
	err := r.store.SessionDelete(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return NewError(err, EntitySession)
	}
	return nil
}

func (r *Session) DeleteAllExcept(ctx context.Context, p session.DeleteAllExceptParams) error {
	err := r.store.SessionDeleteAllExcept(ctx, db.SessionDeleteAllExceptParams{
		UserID: pgtype.UUID{Bytes: p.UserID, Valid: true},
		ID:     pgtype.UUID{Bytes: p.SessionID, Valid: true},
	})
	if err != nil {
		return NewError(err, EntitySession)
	}
	return nil
}

func sessionFromDB(row db.Session) session.Session {
	s := session.Session{
		ID:               uuid.UUID(row.ID.Bytes),
		UserID:           uuid.UUID(row.UserID.Bytes),
		RefreshTokenHash: row.RefreshTokenHash,
		ExpiresAt:        row.ExpiresAt.Time,
		ClientIP:         row.ClientIP.String(),
		UserAgent:        row.UserAgent,
		OS:               row.OS,
		Browser:          row.Browser,
		CreatedAt:        row.CreatedAt.Time,
		UpdatedAt:        row.UpdatedAt.Time,
	}

	if row.RevokedAt.Valid {
		s.RevokedAt = ptr.To(row.RevokedAt.Time)
	}

	if row.LastSeenAt.Valid {
		s.LastSeenAt = row.LastSeenAt.Time
	}

	return s
}

var _ session.Repository = (*Session)(nil)
