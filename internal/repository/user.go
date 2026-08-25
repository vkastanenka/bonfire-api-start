package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

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

func (r *UserRepository) Create(ctx context.Context, u *user.User) (*user.User, error) {
	row, err := r.store.UserCreate(ctx, db.UserCreateParams{
		ID:                     db.ToUUID(u.ID()),
		Email:                  u.Email().String(),
		Username:               u.Username().String(),
		DisplayName:            u.DisplayName().String(),
		PasswordHash:           u.PasswordHash().String(),
		Phone:                  db.ToTextPtr(u.Phone().StringPtr()),
		Bio:                    db.ToTextPtr(u.Bio().StringPtr()),
		AvatarURL:              db.ToTextPtr(u.AvatarURL().StringPtr()),
		BannerColor:            db.ToTextPtr(u.BannerColor().StringPtr()),
		PreferredPresence:      db.ToInt2Ptr(u.PreferredPresence().Presence().IntPtr()),
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

func (r *UserRepository) GetBatch(ctx context.Context, ids []fields.ID) (map[fields.ID]*user.User, error) {
	if len(ids) == 0 {
		return make(map[fields.ID]*user.User), nil
	}

	uuids := make([]uuid.UUID, len(ids))
	for i, id := range ids {
		uuids[i] = id.UUID()
	}

	rows, err := r.store.UserGetBatch(ctx, db.ToUUIDs(uuids))
	if err != nil {
		return nil, r.store.Err(err)
	}

	result := make(map[fields.ID]*user.User, len(rows))
	for _, row := range rows {
		u, err := userFromRow(row)
		if err != nil {
			return nil, err
		}
		result[u.ID()] = u
	}

	return result, nil
}

func (r *UserRepository) ListDeleteScheduled(ctx context.Context, currentTime fields.Timestamp, limitVal int) ([]*user.User, error) {
	rows, err := r.store.UserListDeleteScheduled(ctx, db.UserListDeleteScheduledParams{
		Now:      db.ToTimestamptz(currentTime.Time()),
		LimitVal: int32(limitVal),
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

func (r *UserRepository) Update(ctx context.Context, u *user.User) (*user.User, error) {
	row, err := r.store.UserUpdate(ctx, db.UserUpdateParams{
		ID:                     db.ToUUID(u.ID()),
		Email:                  u.Email().String(),
		Username:               u.Username().String(),
		DisplayName:            u.DisplayName().String(),
		PasswordHash:           u.PasswordHash().String(),
		Phone:                  db.ToTextPtr(u.Phone().StringPtr()),
		Bio:                    db.ToTextPtr(u.Bio().StringPtr()),
		AvatarURL:              db.ToTextPtr(u.AvatarURL().StringPtr()),
		BannerColor:            db.ToTextPtr(u.BannerColor().StringPtr()),
		PreferredPresence:      db.ToInt2Ptr(u.PreferredPresence().Presence().IntPtr()),
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

func (r *UserRepository) UpdateEmail(ctx context.Context, id fields.ID, email user.Email, updatedAt fields.Timestamp) (*user.User, error) {
	row, err := r.store.UserUpdateEmail(ctx, db.UserUpdateEmailParams{
		ID:        db.ToUUID(id.UUID()),
		Email:     email.String(),
		UpdatedAt: db.ToTimestamptz(updatedAt.Time()),
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return userFromRow(row)
}

func (r *UserRepository) UpdateUsername(ctx context.Context, id fields.ID, username user.Username, updatedAt fields.Timestamp) (*user.User, error) {
	row, err := r.store.UserUpdateUsername(ctx, db.UserUpdateUsernameParams{
		ID:        db.ToUUID(id.UUID()),
		Username:  username.String(),
		UpdatedAt: db.ToTimestamptz(updatedAt.Time()),
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return userFromRow(row)
}

func (r *UserRepository) UpdatePhone(ctx context.Context, id fields.ID, phone user.Phone, updatedAt fields.Timestamp) (*user.User, error) {
	row, err := r.store.UserUpdatePhone(ctx, db.UserUpdatePhoneParams{
		ID:        db.ToUUID(id.UUID()),
		Phone:     db.ToTextPtr(phone.StringPtr()),
		UpdatedAt: db.ToTimestamptz(updatedAt.Time()),
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return userFromRow(row)
}

func (r *UserRepository) UpdatePasswordHash(ctx context.Context, id fields.ID, passwordHash user.PasswordHash, updatedAt fields.Timestamp) (*user.User, error) {
	row, err := r.store.UserUpdatePasswordHash(ctx, db.UserUpdatePasswordHashParams{
		ID:           db.ToUUID(id.UUID()),
		PasswordHash: passwordHash.String(),
		UpdatedAt:    db.ToTimestamptz(updatedAt.Time()),
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return userFromRow(row)
}

func (r *UserRepository) UpdateProfile(
	ctx context.Context,
	id fields.ID,
	displayName user.DisplayName,
	bio user.Bio,
	avatarURL fields.URL,
	bannerColor fields.HexColor,
	updatedAt fields.Timestamp,
) (*user.User, error) {
	row, err := r.store.UserUpdateProfile(ctx, db.UserUpdateProfileParams{
		ID:          db.ToUUID(id.UUID()),
		DisplayName: displayName.String(),
		Bio:         db.ToTextPtr(bio.StringPtr()),
		AvatarURL:   db.ToTextPtr(avatarURL.StringPtr()),
		BannerColor: db.ToTextPtr(bannerColor.StringPtr()),
		UpdatedAt:   db.ToTimestamptz(updatedAt.Time()),
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return userFromRow(row)
}

func (r *UserRepository) UpdatePresence(
	ctx context.Context,
	id fields.ID,
	presence user.PreferredPresence,
	presenceUntil fields.Timestamp,
	updatedAt fields.Timestamp,
) (*user.User, error) {
	row, err := r.store.UserUpdatePresence(ctx, db.UserUpdatePresenceParams{
		ID:                     db.ToUUID(id.UUID()),
		PreferredPresence:      db.ToInt2Ptr(presence.Presence().IntPtr()),
		PreferredPresenceUntil: db.ToTimestamptzPtr(presenceUntil.TimePtr()),
		UpdatedAt:              db.ToTimestamptz(updatedAt.Time()),
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return userFromRow(row)
}

func (r *UserRepository) Verify(ctx context.Context, id fields.ID, verifiedAt, updatedAt fields.Timestamp) (*user.User, error) {
	row, err := r.store.UserVerify(ctx, db.UserVerifyParams{
		ID:         db.ToUUID(id.UUID()),
		VerifiedAt: db.ToTimestamptzPtr(verifiedAt.TimePtr()),
		UpdatedAt:  db.ToTimestamptz(updatedAt.Time()),
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return userFromRow(row)
}

func (r *UserRepository) SetDisabled(ctx context.Context, id fields.ID, disabledAt, updatedAt fields.Timestamp) (*user.User, error) {
	row, err := r.store.UserSetDisabled(ctx, db.UserSetDisabledParams{
		ID:         db.ToUUID(id.UUID()),
		DisabledAt: db.ToTimestamptzPtr(disabledAt.TimePtr()),
		UpdatedAt:  db.ToTimestamptz(updatedAt.Time()),
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return userFromRow(row)
}

func (r *UserRepository) SetDeleteSchedule(ctx context.Context, id fields.ID, deleteScheduledAt, disabledAt, updatedAt fields.Timestamp) (*user.User, error) {
	row, err := r.store.UserSetDeleteSchedule(ctx, db.UserSetDeleteScheduleParams{
		ID:                db.ToUUID(id.UUID()),
		DeleteScheduledAt: db.ToTimestamptzPtr(deleteScheduledAt.TimePtr()),
		DisabledAt:        db.ToTimestamptzPtr(disabledAt.TimePtr()),
		UpdatedAt:         db.ToTimestamptz(updatedAt.Time()),
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return userFromRow(row)
}

func (r *UserRepository) UpdateBatch(ctx context.Context, users []*user.User) ([]*user.User, error) {
	if len(users) == 0 {
		return []*user.User{}, nil
	}

	type userPayload struct {
		ID                     uuid.UUID  `json:"id"`
		Email                  string     `json:"email"`
		Username               string     `json:"username"`
		DisplayName            string     `json:"display_name"`
		PasswordHash           string     `json:"password_hash"`
		Phone                  *string    `json:"phone,omitempty"`
		Bio                    *string    `json:"bio,omitempty"`
		AvatarURL              *string    `json:"avatar_url,omitempty"`
		BannerColor            *string    `json:"banner_color,omitempty"`
		PreferredPresence      *int16     `json:"preferred_presence,omitempty"`
		PreferredPresenceUntil *time.Time `json:"preferred_presence_until,omitempty"`
		VerifiedAt             *time.Time `json:"verified_at,omitempty"`
		DisabledAt             *time.Time `json:"disabled_at,omitempty"`
		DeleteScheduledAt      *time.Time `json:"delete_scheduled_at,omitempty"`
		CreatedAt              time.Time  `json:"created_at"`
		UpdatedAt              time.Time  `json:"updated_at"`
	}

	payloads := make([]userPayload, len(users))
	for i, u := range users {
		payloads[i] = userPayload{
			ID:                     u.ID().UUID(),
			Email:                  u.Email().String(),
			Username:               u.Username().String(),
			DisplayName:            u.DisplayName().String(),
			PasswordHash:           u.PasswordHash().String(),
			Phone:                  u.Phone().StringPtr(),
			Bio:                    u.Bio().StringPtr(),
			AvatarURL:              u.AvatarURL().StringPtr(),
			BannerColor:            u.BannerColor().StringPtr(),
			PreferredPresence:      u.PreferredPresence().Presence().Int16Ptr(),
			PreferredPresenceUntil: u.PreferredPresenceUntil().TimePtr(),
			VerifiedAt:             u.VerifiedAt().TimePtr(),
			DisabledAt:             u.DisabledAt().TimePtr(),
			DeleteScheduledAt:      u.DeleteScheduledAt().TimePtr(),
			CreatedAt:              u.CreatedAt().Time(),
			UpdatedAt:              u.UpdatedAt().Time(),
		}
	}

	jsonBytes, err := json.Marshal(payloads)
	if err != nil {
		return nil, errs.Internal("failed to marshal user update batch payload").
			Meta("scope", db.EntityUser.String()).
			Wrap(err)
	}

	rows, err := r.store.UserUpdateBatch(ctx, jsonBytes)
	if err != nil {
		return nil, r.store.Err(err)
	}

	updatedUsers := make([]*user.User, len(rows))
	for i, row := range rows {
		u, err := userFromRow(row)
		if err != nil {
			return nil, err
		}
		updatedUsers[i] = u
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

	avatarURL, err := fields.ParseURL("avatar_url", db.FromText[string](row.AvatarURL))
	if err != nil {
		return nil, mapErr("failed to parse avatar url from database", "avatar_url", row.AvatarURL.String, err)
	}

	bannerColor, err := fields.ParseHexColor("banner_color", db.FromText[string](row.BannerColor))
	if err != nil {
		return nil, mapErr("failed to parse banner color from database", "banner_color", row.BannerColor.String, err)
	}

	rowPreferredPresence := db.FromInt2[int16](row.PreferredPresence)
	preferredPresence, err := user.ParsePreferredPresenceFromInt("preferred_presence", rowPreferredPresence)
	if err != nil {
		return nil, mapErr("failed to parse preferred presence from database", "preferred_presence", rowPreferredPresence, err)
	}

	preferredPresenceUntil := fields.NewTimestamp(db.FromTimestamptz(row.PreferredPresenceUntil))
	verifiedAt := fields.NewTimestamp(db.FromTimestamptz(row.VerifiedAt))
	disabledAt := fields.NewTimestamp(db.FromTimestamptz(row.DisabledAt))
	deleteScheduledAt := fields.NewTimestamp(db.FromTimestamptz(row.DeleteScheduledAt))
	createdAt := fields.NewTimestamp(db.FromTimestamptz(row.CreatedAt))
	updatedAt := fields.NewTimestamp(db.FromTimestamptz(row.UpdatedAt))

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
