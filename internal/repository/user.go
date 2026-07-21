package repository

import (
	"context"
	"time"

	"bonfire-api/internal/apperr"
	"bonfire-api/internal/db"
	"bonfire-api/internal/user"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type User struct {
	q db.Querier
}

func NewUser(q db.Querier) *User {
	return &User{q: q}
}

func (r *User) Create(ctx context.Context, u *user.User) error {
	prof := u.Profile()

	err := r.q.UserCreateAggregate(ctx, db.UserCreateAggregateParams{
		UserID:            pgtype.UUID{Bytes: u.ID(), Valid: true},
		Email:             u.Email().String(),
		Username:          u.Username().String(),
		PasswordHash:      u.PasswordHash(),
		PreferredPresence: pgtype.Int2{Int16: int16(u.PreferredPresence()), Valid: u.PreferredPresence() != 0},
		VerifiedAt:        pgtype.Timestamptz{Time: ptrValue(u.VerifiedAt()), Valid: u.VerifiedAt() != nil},
		UserCreatedAt:     pgtype.Timestamptz{Time: u.CreatedAt(), Valid: true},
		UserUpdatedAt:     pgtype.Timestamptz{Time: u.UpdatedAt(), Valid: true},
		DisplayName:       prof.DisplayName().String(),
		AvatarUrl:         pgtype.Text{String: ptrValue(prof.AvatarURL()), Valid: prof.AvatarURL() != nil},
		ProfileCreatedAt:  pgtype.Timestamptz{Time: prof.CreatedAt(), Valid: true},
		ProfileUpdatedAt:  pgtype.Timestamptz{Time: prof.UpdatedAt(), Valid: true},
	})
	if err != nil {
		return db.NewError(err, db.EntityUser)
	}

	return nil
}

func (r *User) Save(ctx context.Context, u *user.User) error {
	_, err := r.q.UserUpdate(ctx, db.UserUpdateParams{
		ID:                pgtype.UUID{Bytes: u.ID(), Valid: true},
		PasswordHash:      u.PasswordHash(),
		PreferredPresence: pgtype.Int2{Int16: int16(u.PreferredPresence()), Valid: u.PreferredPresence() != 0},
		VerifiedAt:        pgtype.Timestamptz{Time: ptrValue(u.VerifiedAt()), Valid: u.VerifiedAt() != nil},
		UpdatedAt:         pgtype.Timestamptz{Time: u.UpdatedAt(), Valid: true},
	})
	if err != nil {
		return db.NewError(err, db.EntityUser)
	}

	return nil
}

func (r *User) SaveProfile(ctx context.Context, u *user.User) error {
	prof := u.Profile()
	_, err := r.q.UserProfileUpsert(ctx, db.UserProfileUpsertParams{
		UserID:      pgtype.UUID{Bytes: u.ID(), Valid: true},
		DisplayName: prof.DisplayName().String(),
		AvatarUrl:   pgtype.Text{String: ptrValue(prof.AvatarURL()), Valid: prof.AvatarURL() != nil},
		CreatedAt:   pgtype.Timestamptz{Time: prof.CreatedAt(), Valid: true},
		UpdatedAt:   pgtype.Timestamptz{Time: prof.UpdatedAt(), Valid: true},
	})
	if err != nil {
		return db.NewError(err, db.EntityUserProfile)
	}

	return nil
}

func (r *User) Get(ctx context.Context, id uuid.UUID) (*user.User, error) {
	row, err := r.q.UserGet(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return nil, db.NewError(err, db.EntityUser)
	}
	return userFromRow(row)
}

func (r *User) GetByEmail(ctx context.Context, email user.Email) (*user.User, error) {
	row, err := r.q.UserGetByEmail(ctx, email.String())
	if err != nil {
		return nil, db.NewError(err, db.EntityUser)
	}
	return userFromRow(row)
}

func (r *User) GetByUsername(ctx context.Context, username user.Username) (*user.User, error) {
	row, err := r.q.UserGetByUsername(ctx, username.String())
	if err != nil {
		return nil, db.NewError(err, db.EntityUser)
	}
	return userFromRow(row)
}

func (r *User) CheckAvailability(ctx context.Context, email user.Email, username user.Username) (user.CheckAvailabilityResult, error) {
	row, err := r.q.UserCheckAvailability(ctx, db.UserCheckAvailabilityParams{
		Email:    email.String(),
		Username: username.String(),
	})
	if err != nil {
		return user.CheckAvailabilityResult{}, db.NewError(err, db.EntityUser)
	}

	return user.CheckAvailabilityResult{
		EmailAvailable:    row.EmailAvailable.Bool,
		UsernameAvailable: row.UsernameAvailable.Bool,
	}, nil
}

func userFromRow(row db.UserAggregate) (*user.User, error) {
	email, err := user.NewEmail(row.Email)
	if err != nil {
		return nil, apperr.NewInternal(err)
	}

	username, err := user.NewUsername(row.Username)
	if err != nil {
		return nil, apperr.NewInternal(err)
	}

	displayName, err := user.NewProfileDisplayName(row.DisplayName)
	if err != nil {
		return nil, apperr.NewInternal(err)
	}

	var avatarURL *string
	if row.AvatarUrl.Valid {
		avatarURL = &row.AvatarUrl.String
	}

	var verifiedAt *time.Time
	if row.VerifiedAt.Valid {
		verifiedAt = &row.VerifiedAt.Time
	}

	profile := user.ReconstituteProfile(
		displayName,
		avatarURL,
		row.ProfileCreatedAt.Time,
		row.ProfileUpdatedAt.Time,
	)

	return user.Reconstitute(
		uuid.UUID(row.ID.Bytes),
		email,
		username,
		row.PasswordHash,
		user.Presence(row.PreferredPresence.Int16),
		verifiedAt,
		row.CreatedAt.Time,
		row.UpdatedAt.Time,
		profile,
	), nil
}

func ptrValue[T any](p *T) T {
	if p == nil {
		var zero T
		return zero
	}
	return *p
}
