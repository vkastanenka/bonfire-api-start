package repository

import (
	"context"

	"bonfire-api/internal/db"
	"bonfire-api/internal/errs"
	"bonfire-api/internal/presence"
	"bonfire-api/internal/user"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type UserStore interface {
	UserCreateAggregate(ctx context.Context, arg db.UserCreateAggregateParams) error
	UserGet(ctx context.Context, id pgtype.UUID) (db.UserAggregate, error)
	UserGetByEmail(ctx context.Context, email string) (db.UserAggregate, error)
	UserGetByUsername(ctx context.Context, username string) (db.UserAggregate, error)
	UserCheckAvailability(ctx context.Context, arg db.UserCheckAvailabilityParams) (db.UserCheckAvailabilityRow, error)
	UserUpdate(ctx context.Context, arg db.UserUpdateParams) (db.User, error)
	UserProfileUpsert(ctx context.Context, arg db.UserProfileUpsertParams) (db.UserProfile, error)
}

type User struct {
	store UserStore
}

func NewUser(store UserStore) *User {
	return &User{store: store}
}

func (r *User) Create(ctx context.Context, u *user.User) error {
	prof := u.Profile()

	err := r.store.UserCreateAggregate(ctx, db.UserCreateAggregateParams{
		UserID:            db.UUID(u.ID()),
		Email:             u.Email().String(),
		Username:          u.Username().String(),
		PasswordHash:      u.PasswordHash(),
		PreferredPresence: db.Int2Ptr(u.PreferredPresence()),
		VerifiedAt:        db.TimestamptzPtr(u.VerifiedAt()),
		UserCreatedAt:     db.Timestamptz(u.CreatedAt()),
		UserUpdatedAt:     db.Timestamptz(u.UpdatedAt()),
		DisplayName:       prof.DisplayName().String(),
		AvatarUrl:         db.Text(prof.AvatarURL()),
		ProfileCreatedAt:  db.Timestamptz(prof.CreatedAt()),
		ProfileUpdatedAt:  db.Timestamptz(prof.UpdatedAt()),
	})
	if err != nil {
		return db.NewError(err, db.EntityUser)
	}

	return nil
}

func (r *User) Get(ctx context.Context, id uuid.UUID) (*user.User, error) {
	row, err := r.store.UserGet(ctx, db.UUID(id))
	if err != nil {
		return nil, db.NewError(err, db.EntityUser)
	}
	return userFromRow(row)
}

func (r *User) GetByEmail(ctx context.Context, email user.Email) (*user.User, error) {
	row, err := r.store.UserGetByEmail(ctx, email.String())
	if err != nil {
		return nil, db.NewError(err, db.EntityUser)
	}
	return userFromRow(row)
}

func (r *User) GetByUsername(ctx context.Context, username user.Username) (*user.User, error) {
	row, err := r.store.UserGetByUsername(ctx, username.String())
	if err != nil {
		return nil, db.NewError(err, db.EntityUser)
	}
	return userFromRow(row)
}

func (r *User) CheckAvailability(ctx context.Context, email user.Email, username user.Username) (bool, bool, error) {
	row, err := r.store.UserCheckAvailability(ctx, db.UserCheckAvailabilityParams{
		Email:    email.String(),
		Username: username.String(),
	})
	if err != nil {
		return false, false, db.NewError(err, db.EntityUser)
	}

	return row.EmailAvailable.Bool, row.UsernameAvailable.Bool, nil
}

func (r *User) Update(ctx context.Context, u *user.User) error {
	_, err := r.store.UserUpdate(ctx, db.UserUpdateParams{
		ID:                db.UUID(u.ID()),
		Email:             u.Email().String(),
		Username:          u.Username().String(),
		PasswordHash:      u.PasswordHash(),
		PreferredPresence: db.Int2Ptr(u.PreferredPresence()),
		VerifiedAt:        db.TimestamptzPtr(u.VerifiedAt()),
		UpdatedAt:         db.Timestamptz(u.UpdatedAt()),
	})
	if err != nil {
		return db.NewError(err, db.EntityUser)
	}

	return nil
}

func (r *User) UpsertProfile(ctx context.Context, userID uuid.UUID, prof *user.Profile) error {
	_, err := r.store.UserProfileUpsert(ctx, db.UserProfileUpsertParams{
		UserID:      db.UUID(userID),
		CreatedAt:   db.Timestamptz(prof.CreatedAt()),
		UpdatedAt:   db.Timestamptz(prof.UpdatedAt()),
		DisplayName: prof.DisplayName().String(),
		AvatarUrl:   db.Text(prof.AvatarURL()),
	})
	if err != nil {
		return db.NewError(err, db.EntityUserProfile)
	}
	return nil
}

func userFromRow(row db.UserAggregate) (*user.User, error) {
	userID := uuid.UUID(row.ID.Bytes).String()

	email, err := user.NewEmail(row.Email)
	if err != nil {
		return nil, errs.Internal("failed to parse user email from database").
			Wrap(err).
			Reason("CORRUPT_DATABASE_RECORD").
			Meta("email", row.Email).
			Resource("User", userID, "", "database aggregate row mapping")
	}

	username, err := user.NewUsername(row.Username)
	if err != nil {
		return nil, errs.Internal("failed to parse username from database").
			Wrap(err).
			Reason("CORRUPT_DATABASE_RECORD").
			Meta("username", row.Username).
			Resource("User", userID, "", "database aggregate row mapping")
	}

	displayName, err := user.NewProfileDisplayName(row.DisplayName)
	if err != nil {
		return nil, errs.Internal("failed to parse profile display name from database").
			Wrap(err).
			Reason("CORRUPT_DATABASE_RECORD").
			Meta("display_name", row.DisplayName).
			Resource("UserProfile", userID, "", "database aggregate row mapping")
	}

	profile := user.ReconstituteProfile(
		displayName,
		db.StringPtr(row.AvatarUrl),
		row.ProfileCreatedAt.Time.UTC(),
		row.ProfileUpdatedAt.Time.UTC(),
	)

	return user.Reconstitute(
		uuid.UUID(row.ID.Bytes),
		email,
		username,
		row.PasswordHash,
		db.Int16Ptr[presence.Presence](row.PreferredPresence),
		db.TimePtr(row.VerifiedAt),
		row.CreatedAt.Time.UTC(),
		row.UpdatedAt.Time.UTC(),
		profile,
	), nil
}
