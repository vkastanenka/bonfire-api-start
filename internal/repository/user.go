package repository

import (
	"context"
	"time"

	"bonfire-api/internal/db"
	"bonfire-api/internal/errs"
	"bonfire-api/internal/presence"
	"bonfire-api/internal/user"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type User struct {
	q db.Querier
}

var _ user.Repository = (*User)(nil)

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
		PreferredPresence: presenceToInt2(u.PreferredPresence()),
		VerifiedAt:        timeToTimestamptz(u.VerifiedAt()),
		UserCreatedAt:     pgtype.Timestamptz{Time: u.CreatedAt(), Valid: true},
		UserUpdatedAt:     pgtype.Timestamptz{Time: u.UpdatedAt(), Valid: true},
		DisplayName:       prof.DisplayName().String(),
		AvatarUrl:         stringPtrToText(prof.AvatarURL()),
		ProfileCreatedAt:  pgtype.Timestamptz{Time: prof.CreatedAt(), Valid: true},
		ProfileUpdatedAt:  pgtype.Timestamptz{Time: prof.UpdatedAt(), Valid: true},
	})
	if err != nil {
		return db.NewError(err, db.EntityUser)
	}

	return nil
}

func (r *User) Save(ctx context.Context, u *user.User) error {
	_, err := r.q.UserSave(ctx, db.UserSaveParams{
		ID:                pgtype.UUID{Bytes: u.ID(), Valid: true},
		PasswordHash:      u.PasswordHash(),
		PreferredPresence: presenceToInt2(u.PreferredPresence()),
		VerifiedAt:        timeToTimestamptz(u.VerifiedAt()),
		UpdatedAt:         pgtype.Timestamptz{Time: u.UpdatedAt(), Valid: true},
	})
	if err != nil {
		return db.NewError(err, db.EntityUser)
	}

	return nil
}

func (r *User) SaveProfile(ctx context.Context, u *user.User) error {
	prof := u.Profile()
	_, err := r.q.UserProfileSave(ctx, db.UserProfileSaveParams{
		UserID:      pgtype.UUID{Bytes: u.ID(), Valid: true},
		DisplayName: prof.DisplayName().String(),
		AvatarUrl:   stringPtrToText(prof.AvatarURL()),
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

func (r *User) CheckAvailability(ctx context.Context, email user.Email, username user.Username) (bool, bool, error) {
	row, err := r.q.UserCheckAvailability(ctx, db.UserCheckAvailabilityParams{
		Email:    email.String(),
		Username: username.String(),
	})
	if err != nil {
		return false, false, db.NewError(err, db.EntityUser)
	}

	return row.EmailAvailable.Bool, row.UsernameAvailable.Bool, nil
}

func userFromRow(row db.UserAggregate) (*user.User, error) {
	email, err := user.NewEmail(row.Email)
	if err != nil {
		return nil, errs.Internal("").Wrap(err)
	}

	username, err := user.NewUsername(row.Username)
	if err != nil {
		return nil, errs.Internal("").Wrap(err)
	}

	displayName, err := user.NewProfileDisplayName(row.DisplayName)
	if err != nil {
		return nil, errs.Internal("").Wrap(err)
	}

	var avatarURL *string
	if row.AvatarUrl.Valid {
		avatarURL = &row.AvatarUrl.String
	}

	var verifiedAt *time.Time
	if row.VerifiedAt.Valid {
		verifiedAt = &row.VerifiedAt.Time
	}

	var preferredPresence *presence.Presence
	if row.PreferredPresence.Valid {
		p := presence.Presence(row.PreferredPresence.Int16)
		preferredPresence = &p
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
		preferredPresence,
		verifiedAt,
		row.CreatedAt.Time,
		row.UpdatedAt.Time,
		profile,
	), nil
}

func presenceToInt2(p *presence.Presence) pgtype.Int2 {
	if p == nil {
		return pgtype.Int2{Valid: false}
	}
	return pgtype.Int2{Int16: int16(*p), Valid: true}
}

func stringPtrToText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func timeToTimestamptz(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}
