package repository

import (
	"context"

	"bonfire-api/internal/db"
	"bonfire-api/internal/errs"
	"bonfire-api/internal/user"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type UserStore interface {
	UserAvailability(ctx context.Context, arg db.UserAvailabilityParams) (db.UserAvailabilityRow, error)
	UserCreate(ctx context.Context, arg db.UserCreateParams) error
	UserGet(ctx context.Context, id pgtype.UUID) (db.User, error)
	UserGetByEmail(ctx context.Context, email string) (db.User, error)
	UserListDeleteScheduled(ctx context.Context, arg db.UserListDeleteScheduledParams) ([]db.User, error)
	UserUpdate(ctx context.Context, arg db.UserUpdateParams) (db.User, error)
	UserUpdateBatch(ctx context.Context, usersJson []byte) ([]db.User, error)
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
		ID:                     db.ToUUID(u.ID()),
		Email:                  u.Email().String(),
		Username:               u.Username().String(),
		DisplayName:            u.DisplayName().String(),
		PasswordHash:           u.PasswordHash().String(),
		Phone:                  db.ToTextPtr(u.Phone().StringPtr()),
		Bio:                    db.ToTextPtr(u.Bio().StringPtr()),
		AvatarUrl:              db.ToTextPtr(u.AvatarURL().StringPtr()),
		BannerColor:            db.ToTextPtr(u.BannerColor().StringPtr()),
		PreferredPresence:      db.ToInt2Ptr(u.PreferredPresence().Int16Ptr()),
		PreferredPresenceUntil: db.ToTimestamptzPtr(u.PreferredPresenceUntil().TimePtr()),
		VerifiedAt:             db.ToTimestamptzPtr(u.VerifiedAt().TimePtr()),
		DisabledAt:             db.ToTimestamptzPtr(u.DisabledAt().TimePtr()),
		DeleteScheduledAt:      db.ToTimestamptzPtr(u.DeleteScheduledAt().TimePtr()),
		CreatedAt:              db.ToTimestamptz(u.CreatedAt().Time()),
		UpdatedAt:              db.ToTimestamptz(u.UpdatedAt().Time()),
	})
	if err != nil {
		return db.NewError(err, db.EntityUser)
	}

	return nil
}

func (r *User) Get(ctx context.Context, id user.ID) (*user.User, error) {
	row, err := r.store.UserGet(ctx, db.ToUUID(id.UUID()))
	if err != nil {
		return nil, db.NewError(err, db.EntityUser)
	}

	return userFromRow(db.User(row))
}

func (r *User) GetByEmail(ctx context.Context, email user.Email) (*user.User, error) {
	row, err := r.store.UserGetByEmail(ctx, email.String())
	if err != nil {
		return nil, db.NewError(err, db.EntityUser)
	}

	return userFromRow(row)
}

func (r *User) ListDeleteScheduled(ctx context.Context, currentTime user.Timestamp, batchLimit int32) ([]*user.User, error) {
	rows, err := r.store.UserListDeleteScheduled(ctx, db.UserListDeleteScheduledParams{
		Now:        db.ToTimestamptz(currentTime.Time()),
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

func (r *User) Update(ctx context.Context, u *user.User) (*user.User, error) {
	row, err := r.store.UserUpdate(ctx, db.UserUpdateParams{
		ID:                     db.ToUUID(u.ID()),
		Email:                  u.Email().String(),
		Username:               u.Username().String(),
		DisplayName:            u.DisplayName().String(),
		PasswordHash:           u.PasswordHash().String(),
		Phone:                  db.ToTextPtr(u.Phone().StringPtr()),
		Bio:                    db.ToTextPtr(u.Bio().StringPtr()),
		AvatarUrl:              db.ToTextPtr(u.AvatarURL().StringPtr()),
		BannerColor:            db.ToTextPtr(u.BannerColor().StringPtr()),
		PreferredPresence:      db.ToInt2Ptr(u.PreferredPresence().Int16Ptr()),
		PreferredPresenceUntil: db.ToTimestamptzPtr(u.PreferredPresenceUntil().TimePtr()),
		VerifiedAt:             db.ToTimestamptzPtr(u.VerifiedAt().TimePtr()),
		DisabledAt:             db.ToTimestamptzPtr(u.DisabledAt().TimePtr()),
		DeleteScheduledAt:      db.ToTimestamptzPtr(u.DeleteScheduledAt().TimePtr()),
		UpdatedAt:              db.ToTimestamptz(u.UpdatedAt().Time()),
	})
	if err != nil {
		return nil, db.NewError(err, db.EntityUser)
	}

	return userFromRow(row)
}

func (r *User) UpdateBatch(ctx context.Context, usersJson []byte) ([]*user.User, error) {
	rows, err := r.store.UserUpdateBatch(ctx, usersJson)
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
	userID := db.FromUUID[uuid.UUID](row.ID)

	id, err := user.NewID(userID)
	if err != nil {
		return nil, errs.Internal("failed to parse user id from database").
			Wrap(err).
			Reason("CORRUPT_DATABASE_RECORD").
			Meta("id", userID.String()).
			Resource("User", userID.String(), "", "database row mapping")
	}

	email, err := user.NewEmail(row.Email)
	if err != nil {
		return nil, errs.Internal("failed to parse user email from database").
			Wrap(err).
			Reason("CORRUPT_DATABASE_RECORD").
			Meta("email", row.Email).
			Resource("User", userID.String(), "", "database row mapping")
	}

	username, err := user.NewUsername(row.Username)
	if err != nil {
		return nil, errs.Internal("failed to parse username from database").
			Wrap(err).
			Reason("CORRUPT_DATABASE_RECORD").
			Meta("username", row.Username).
			Resource("User", userID.String(), "", "database row mapping")
	}

	passwordHash, err := user.NewPasswordHash(row.PasswordHash)
	if err != nil {
		return nil, errs.Internal("failed to parse password hash from database").
			Wrap(err).
			Reason("CORRUPT_DATABASE_RECORD").
			Meta("password_hash", row.PasswordHash).
			Resource("User", userID.String(), "", "database row mapping")
	}

	displayName, err := user.NewDisplayName(row.DisplayName)
	if err != nil {
		return nil, errs.Internal("failed to parse display name from database").
			Wrap(err).
			Reason("CORRUPT_DATABASE_RECORD").
			Meta("display_name", row.DisplayName).
			Resource("User", userID.String(), "", "database row mapping")
	}

	phone, err := user.NewPhone(db.FromText[string](row.Phone))
	if err != nil {
		return nil, errs.Internal("failed to parse phone from database").
			Wrap(err).
			Reason("CORRUPT_DATABASE_RECORD").
			Meta("phone", row.Phone.String).
			Resource("User", userID.String(), "", "database row mapping")
	}

	bio, err := user.NewBio(db.FromText[string](row.Bio))
	if err != nil {
		return nil, errs.Internal("failed to parse bio from database").
			Wrap(err).
			Reason("CORRUPT_DATABASE_RECORD").
			Meta("bio", row.Bio.String).
			Resource("User", userID.String(), "", "database row mapping")
	}

	avatarURL, err := user.NewURL(db.FromText[string](row.AvatarUrl))
	if err != nil {
		return nil, errs.Internal("failed to parse avatar url from database").
			Wrap(err).
			Reason("CORRUPT_DATABASE_RECORD").
			Meta("avatar_url", row.AvatarUrl.String).
			Resource("User", userID.String(), "", "database row mapping")
	}

	bannerColor, err := user.NewHexColor(db.FromText[string](row.BannerColor))
	if err != nil {
		return nil, errs.Internal("failed to parse banner color from database").
			Wrap(err).
			Reason("CORRUPT_DATABASE_RECORD").
			Meta("banner_color", row.BannerColor.String).
			Resource("User", userID.String(), "", "database row mapping")
	}

	rowPreferredPresence := db.FromInt2[int16](row.PreferredPresence)
	preferredPresence, err := user.PreferredPresenceFromInt16(rowPreferredPresence)
	if err != nil {
		return nil, errs.Internal("failed to parse preferred presence from database").
			Wrap(err).
			Reason("CORRUPT_DATABASE_RECORD").
			Meta("preferred_presence", string(rowPreferredPresence)).
			Resource("User", userID.String(), "", "database row mapping")
	}

	preferredPresenceUntil := user.NewTimestampFromTime(db.FromTimestamptz(row.PreferredPresenceUntil))
	verifiedAt := user.NewTimestampFromTime(db.FromTimestamptz(row.VerifiedAt))
	disabledAt := user.NewTimestampFromTime(db.FromTimestamptz(row.DisabledAt))
	deleteScheduledAt := user.NewTimestampFromTime(db.FromTimestamptz(row.DeleteScheduledAt))
	createdAt := user.NewTimestampFromTime(db.FromTimestamptz(row.CreatedAt))
	updatedAt := user.NewTimestampFromTime(db.FromTimestamptz(row.UpdatedAt))

	return user.Reconstitute(
		id,
		email,
		username,
		passwordHash,
		phone,
		displayName,
		bio,
		avatarURL,
		bannerColor,
		preferredPresence,
		preferredPresenceUntil,
		verifiedAt,
		disabledAt,
		deleteScheduledAt,
		createdAt,
		updatedAt,
	), nil
}
