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

type UserStore interface {
	UserAvailability(ctx context.Context, arg db.UserAvailabilityParams) (db.UserAvailabilityRow, error)
	UserCreate(ctx context.Context, arg db.UserCreateParams) error
	UserGet(ctx context.Context, id pgtype.UUID) (db.UserGetRow, error)
	UserGetByEmail(ctx context.Context, email string) (db.UserGetByEmailRow, error)
	UserListDeleteScheduled(ctx context.Context, arg db.UserListDeleteScheduledParams) ([]db.UserListDeleteScheduledRow, error)
	UserUpdate(ctx context.Context, arg db.UserUpdateParams) (db.UserUpdateRow, error)
	UserUpdateBatch(ctx context.Context, usersJson []byte) ([]db.UserUpdateBatchRow, error)
}

type User struct {
	store UserStore
}

func NewUser(store UserStore) *User {
	return &User{store: store}
}

func (r *User) Availability(ctx context.Context, email user.Email, username user.Username) (bool, bool, error) {
	row, err := r.store.UserAvailability(ctx, db.UserAvailabilityParams{
		Email:    email.String(),
		Username: username.String(),
	})
	if err != nil {
		return false, false, db.NewError(err, db.EntityUser)
	}
	return row.EmailAvailable.Bool, row.UsernameAvailable.Bool, nil
}

func (r *User) Create(ctx context.Context, u *user.User) error {
	err := r.store.UserCreate(ctx, db.UserCreateParams{
		ID:                     db.UUID(u.ID()),
		Email:                  u.Email().String(),
		Username:               u.Username().String(),
		DisplayName:            u.DisplayName().String(),
		PasswordHash:           u.PasswordHash().String(),
		Phone:                  db.TextPtr(u.Phone().NilString()),
		Bio:                    db.TextPtr(u.Bio().NilString()),
		AvatarUrl:              db.TextPtr(u.AvatarURL().NilString()),
		BannerColor:            db.TextPtr(u.BannerColor().NilString()),
		PreferredPresence:      db.Int2Ptr(u.PreferredPresence().NilPresence()),
		PreferredPresenceUntil: db.TimestamptzPtr(u.PreferredPresenceUntil().Time()),
		VerifiedAt:             db.TimestamptzPtr(u.VerifiedAt().Time()),
		DisabledAt:             db.TimestamptzPtr(u.DisabledAt().Time()),
		DeleteScheduledAt:      db.TimestamptzPtr(u.DeleteScheduledAt().Time()),
		CreatedAt:              db.Timestamptz(*u.CreatedAt().Time()),
		UpdatedAt:              db.Timestamptz(*u.UpdatedAt().Time()),
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

func (r *User) ListDeleteScheduled(ctx context.Context, currentTime time.Time, batchLimit int32) ([]*user.User, error) {
	rows, err := r.store.UserListDeleteScheduled(ctx, db.UserListDeleteScheduledParams{
		Now:        db.Timestamptz(currentTime),
		BatchLimit: batchLimit,
	})
	if err != nil {
		return nil, db.NewError(err, db.EntityUser)
	}

	users := make([]*user.User, 0, len(rows))
	for _, row := range rows {
		u, err := userFromRow(row)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}

	return users, nil
}

func (r *User) Update(ctx context.Context, u *user.User) error {
	prof := u.Profile()

	_, err := r.store.UserUpdate(ctx, db.UserUpdateParams{
		ID:                     db.UUID(u.ID()),
		Email:                  u.Email().String(),
		Username:               u.Username().String(),
		DisplayName:            prof.DisplayName().String(),
		PasswordHash:           u.PasswordHash(),
		Phone:                  db.TextPtr(u.Phone()),
		Bio:                    db.TextPtr(prof.Bio()),
		AvatarUrl:              db.TextPtr(prof.AvatarURL()),
		BannerColor:            db.TextPtr(prof.BannerColor()),
		PreferredPresence:      db.Int2Ptr(u.PreferredPresence()),
		PreferredPresenceUntil: db.TimestamptzPtr(u.PreferredPresenceUntil()),
		VerifiedAt:             db.TimestamptzPtr(u.VerifiedAt()),
		DisabledAt:             db.TimestamptzPtr(u.DisabledAt()),
		DeleteScheduledAt:      db.TimestamptzPtr(u.DeleteScheduledAt()),
		UpdatedAt:              db.Timestamptz(u.UpdatedAt()),
	})
	if err != nil {
		return db.NewError(err, db.EntityUser)
	}

	return nil
}

func (r *User) UpdateBatch(ctx context.Context, usersJson []byte) ([]*user.User, error) {
	rows, err := r.store.UserUpdateBatch(ctx, db.UserUpdateBatchParams{
		UsersJson: usersJson,
	})
	if err != nil {
		return nil, db.NewError(err, db.EntityUser)
	}

	updatedUsers := make([]*user.User, 0, len(rows))
	for _, row := range rows {
		u, err := userFromRow(row)
		if err != nil {
			return nil, err
		}
		updatedUsers = append(updatedUsers, u)
	}

	return updatedUsers, nil
}

func userFromRow(row db.User) (*user.User, error) {
	userID := uuid.UUID(row.ID.Bytes).String()

	email, err := user.NewEmail(row.Email)
	if err != nil {
		return nil, errs.Internal("failed to parse user email from database").
			Wrap(err).
			Reason("CORRUPT_DATABASE_RECORD").
			Meta("email", row.Email).
			Resource("User", userID, "", "database row mapping")
	}

	username, err := user.NewUsername(row.Username)
	if err != nil {
		return nil, errs.Internal("failed to parse username from database").
			Wrap(err).
			Reason("CORRUPT_DATABASE_RECORD").
			Meta("username", row.Username).
			Resource("User", userID, "", "database row mapping")
	}

	displayName, err := user.NewProfileDisplayName(row.DisplayName)
	if err != nil {
		return nil, errs.Internal("failed to parse display name from database").
			Wrap(err).
			Reason("CORRUPT_DATABASE_RECORD").
			Meta("display_name", row.DisplayName).
			Resource("User", userID, "", "database row mapping")
	}

	profile := user.ReconstituteProfile(
		displayName,
		db.StringPtr(row.Bio),
		db.StringPtr(row.AvatarUrl),
		db.StringPtr(row.BannerColor),
	)

	return user.Reconstitute(
		uuid.UUID(row.ID.Bytes),
		email,
		username,
		row.PasswordHash,
		db.StringPtr(row.Phone),
		db.Int16Ptr[presence.Presence](row.PreferredPresence),
		db.TimePtr(row.PreferredPresenceUntil),
		db.TimePtr(row.VerifiedAt),
		db.TimePtr(row.DisabledAt),
		db.TimePtr(row.DeleteScheduledAt),
		row.CreatedAt.Time.UTC(),
		row.UpdatedAt.Time.UTC(),
		profile,
	), nil
}
