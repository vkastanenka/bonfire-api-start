package repository

import (
	"context"
	"fmt"

	"bonfire-api/internal/db"
	"bonfire-api/internal/errs"
	"bonfire-api/internal/fields"
	"bonfire-api/internal/user"

	"github.com/google/uuid"
)

type UserRepository struct {
	store *db.Store
}

func NewUserRepository(store *db.Store) *UserRepository {
	return &UserRepository{
		store: store.WithEntity(db.EntityUser),
	}
}

func (r *UserRepository) Availability(ctx context.Context, email *user.Email, username *user.Username) (bool, bool, error) {
	var emailStr, usernameStr string
	if email != nil {
		emailStr = email.String()
	}
	if username != nil {
		usernameStr = username.String()
	}

	row, err := r.store.UserAvailability(ctx, db.UserAvailabilityParams{
		Email:    emailStr,
		Username: usernameStr,
	})
	if err != nil {
		return false, false, r.store.Err(err)
	}

	return row.EmailAvailable.Bool, row.UsernameAvailable.Bool, nil
}

func (r *UserRepository) Create(ctx context.Context, u *user.User) (*user.User, error) {
	row, err := r.store.UserCreate(ctx, db.UserCreateParams{
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
		return nil, r.store.Err(err)
	}

	return userFromRow(row)
}

func (r *UserRepository) Get(ctx context.Context, id fields.ID) (*user.User, error) {
	row, err := r.store.UserGet(ctx, db.ToUUID(id.UUID()))
	if err != nil {
		return nil, r.store.Err(err)
	}

	return userFromRow(row)
}

func (r *UserRepository) GetByEmail(ctx context.Context, email user.Email) (*user.User, error) {
	row, err := r.store.UserGetByEmail(ctx, email.String())
	if err != nil {
		return nil, r.store.Err(err)
	}

	return userFromRow(row)
}

func (r *UserRepository) GetDeleteScheduledBatch(ctx context.Context, currentTime fields.Timestamp, batchLimit int32) ([]*user.User, error) {
	rows, err := r.store.UserGetDeleteScheduledBatch(ctx, db.UserGetDeleteScheduledBatchParams{
		Now:        db.ToTimestamptz(currentTime.Time()),
		BatchLimit: batchLimit,
	})
	if err != nil {
		return nil, r.store.Err(err)
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

func (r *UserRepository) Update(ctx context.Context, u *user.User) (*user.User, error) {
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
		return nil, r.store.Err(err)
	}

	return userFromRow(row)
}

func (r *UserRepository) UpdateBatch(ctx context.Context, usersJson []byte) ([]*user.User, error) {
	rows, err := r.store.UserUpdateBatch(ctx, usersJson)
	if err != nil {
		return nil, r.store.Err(err)
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
	userIDStr := userID.String()

	mapErr := func(msg, key string, val any, err error) *errs.Error {
		return errs.Internal(msg).
			Wrap(err).
			Reason("CORRUPT_DATABASE_RECORD").
			Meta(key, fmt.Sprintf("%v", val)).
			Resource("User", userIDStr, "", "database row mapping")
	}

	id, err := fields.ParseRequiredID("id", userID)
	if err != nil {
		return nil, mapErr("failed to parse user id from database", "id", userIDStr, err)
	}

	email, err := user.ParseEmail("email", row.Email)
	if err != nil {
		return nil, mapErr("failed to parse user email from database", "email", row.Email, err)
	}

	username, err := user.ParseUsername("username", row.Username)
	if err != nil {
		return nil, mapErr("failed to parse username from database", "username", row.Username, err)
	}

	passwordHash, err := user.ParsePasswordHash("password_hash", row.PasswordHash)
	if err != nil {
		return nil, mapErr("failed to parse password hash from database", "password_hash", row.PasswordHash, err)
	}

	displayName, err := user.ParseDisplayName("display_name", row.DisplayName)
	if err != nil {
		return nil, mapErr("failed to parse display name from database", "display_name", row.DisplayName, err)
	}

	phone, err := user.ParsePhone("phone", db.FromText[string](row.Phone))
	if err != nil {
		return nil, mapErr("failed to parse phone from database", "phone", row.Phone.String, err)
	}

	bio, err := user.ParseBio("bio", db.FromText[string](row.Bio))
	if err != nil {
		return nil, mapErr("failed to parse bio from database", "bio", row.Bio.String, err)
	}

	avatarURL, err := fields.ParseURL("avatar_url", db.FromText[string](row.AvatarUrl))
	if err != nil {
		return nil, mapErr("failed to parse avatar url from database", "avatar_url", row.AvatarUrl.String, err)
	}

	bannerColor, err := fields.ParseHexColor("banner_color", db.FromText[string](row.BannerColor))
	if err != nil {
		return nil, mapErr("failed to parse banner color from database", "banner_color", row.BannerColor.String, err)
	}

	rowPreferredPresence := db.FromInt2[int16](row.PreferredPresence)
	preferredPresence, err := user.ParsePreferredPresenceFromInt16("preferred_presence", rowPreferredPresence)
	if err != nil {
		return nil, mapErr("failed to parse preferred presence from database", "preferred_presence", rowPreferredPresence, err)
	}

	preferredPresenceUntil := fields.NewTimestampFromTime(db.FromTimestamptz(row.PreferredPresenceUntil))
	verifiedAt := fields.NewTimestampFromTime(db.FromTimestamptz(row.VerifiedAt))
	disabledAt := fields.NewTimestampFromTime(db.FromTimestamptz(row.DisabledAt))
	deleteScheduledAt := fields.NewTimestampFromTime(db.FromTimestamptz(row.DeleteScheduledAt))
	createdAt := fields.NewTimestampFromTime(db.FromTimestamptz(row.CreatedAt))
	updatedAt := fields.NewTimestampFromTime(db.FromTimestamptz(row.UpdatedAt))

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
