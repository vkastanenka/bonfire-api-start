// internal/db/user_repository.go
package db

import (
	"context"
	"time"

	"bonfire-api/internal/pkg/ptr"
	"bonfire-api/internal/user"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type UserRepository struct {
	store Store
}

func NewUserRepository(store Store) *UserRepository {
	return &UserRepository{store: store}
}

func (r *UserRepository) CheckAvailability(ctx context.Context, email, username string) (bool, bool, error) {
	row, err := r.store.UserCheckAvailability(ctx, UserCheckAvailabilityParams{
		Email:    email,
		Username: username,
	})
	if err != nil {
		return false, false, NewError(err, EntityUser)
	}
	return row.EmailAvailable, row.UsernameAvailable, nil
}

// Save handles both Insert and Update operations for the User aggregate root.
func (r *UserRepository) Save(ctx context.Context, u *user.User) error {
	var verifiedAt pgtype.Timestamptz
	if u.VerifiedAt() != nil {
		verifiedAt = pgtype.Timestamptz{Time: *u.VerifiedAt(), Valid: true}
	}

	presence := pgtype.Int2{Int16: int16(u.Presence()), Valid: true}

	// UserSave should map to an UPSERT SQL query:
	// INSERT INTO users (id, email, username, password_hash, preferred_presence, verified_at, created_at, updated_at)
	// VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	// ON CONFLICT (id) DO UPDATE SET
	//   email = EXCLUDED.email,
	//   username = EXCLUDED.username,
	//   password_hash = EXCLUDED.password_hash,
	//   preferred_presence = EXCLUDED.preferred_presence,
	//   verified_at = EXCLUDED.verified_at,
	//   updated_at = EXCLUDED.updated_at;
	err := r.store.UserSave(ctx, UserSaveParams{
		ID:           pgtype.UUID{Bytes: u.ID(), Valid: true},
		Email:        u.Email(),
		Username:     u.Username(),
		PasswordHash: u.PasswordHash(),
		Presence:     presence,
		VerifiedAt:   verifiedAt,
		CreatedAt:    pgtype.Timestamptz{Time: u.CreatedAt(), Valid: true},
		UpdatedAt:    pgtype.Timestamptz{Time: u.UpdatedAt(), Valid: true},
	})
	if err != nil {
		return NewError(err, EntityUser)
	}
	return nil
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	row, err := r.store.UserGetByID(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return nil, NewError(err, EntityUser)
	}
	return userFromDB(row), nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	row, err := r.store.UserGetByEmail(ctx, email)
	if err != nil {
		return nil, NewError(err, EntityUser)
	}
	return userFromDB(row), nil
}

func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*user.User, error) {
	row, err := r.store.UserGetByUsername(ctx, username)
	if err != nil {
		return nil, NewError(err, EntityUser)
	}
	return userFromDB(row), nil
}

func userFromDB(row User) *user.User {
	var verifiedAt *time.Time
	if row.VerifiedAt.Valid {
		verifiedAt = ptr.To(row.VerifiedAt.Time)
	}

	presence := user.PresenceUnknown
	if row.Presence.Valid {
		presence = user.Presence(row.Presence.Int16)
	}

	return user.Reconstitute(
		uuid.UUID(row.ID.Bytes),
		row.Email,
		row.Username,
		row.PasswordHash,
		presence,
		verifiedAt,
		row.CreatedAt.Time,
		row.UpdatedAt.Time,
	)
}
