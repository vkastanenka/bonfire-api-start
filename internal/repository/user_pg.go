package repository

import (
	"context"
	"encoding/json"
	"time"

	"bonfire-api/internal/db"
	"bonfire-api/internal/errs"
	"bonfire-api/internal/presence"
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
		ID:                     db.ToUUID(u.ID),
		Email:                  u.Email,
		Username:               u.Username,
		DisplayName:            u.DisplayName,
		PasswordHash:           u.PasswordHash,
		Phone:                  db.ToTextPtr(u.Phone),
		Bio:                    db.ToTextPtr(u.Bio),
		AvatarURL:              db.ToTextPtr(u.AvatarURL),
		BannerColor:            db.ToTextPtr(u.BannerColor),
		PreferredPresence:      db.ToInt2Ptr(u.PreferredPresence),
		PreferredPresenceUntil: db.ToTimestamptzPtr(u.PreferredPresenceUntil),
		VerifiedAt:             db.ToTimestamptzPtr(u.VerifiedAt),
		DisabledAt:             db.ToTimestamptzPtr(u.DisabledAt),
		DeleteScheduledAt:      db.ToTimestamptzPtr(u.DeleteScheduledAt),
		CreatedAt:              db.ToTimestamptz(u.CreatedAt),
		UpdatedAt:              db.ToTimestamptz(u.UpdatedAt),
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return userFromRow(row), nil
}

func (r *UserRepository) Get(ctx context.Context, id uuid.UUID) (*user.User, error) {
	row, err := r.store.UserGet(ctx, db.ToUUID(id))
	if err != nil {
		return nil, r.store.Err(err)
	}

	return userFromRow(row), nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	row, err := r.store.UserGetByEmail(ctx, email)
	if err != nil {
		return nil, r.store.Err(err)
	}

	return userFromRow(row), nil
}

func (r *UserRepository) GetBatch(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*user.User, error) {
	if len(ids) == 0 {
		return make(map[uuid.UUID]*user.User), nil
	}

	rows, err := r.store.UserGetBatch(ctx, db.ToUUIDs(ids))
	if err != nil {
		return nil, r.store.Err(err)
	}

	result := make(map[uuid.UUID]*user.User, len(rows))
	for _, row := range rows {
		u := userFromRow(row)
		result[u.ID] = u
	}

	return result, nil
}

func (r *UserRepository) ListDeleteScheduled(ctx context.Context, currentTime time.Time, limitVal int) ([]*user.User, error) {
	rows, err := r.store.UserListDeleteScheduled(ctx, db.UserListDeleteScheduledParams{
		Now:      db.ToTimestamptz(currentTime),
		LimitVal: int32(limitVal),
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	users := make([]*user.User, 0, len(rows))
	for _, row := range rows {
		u := userFromRow(row)
		users = append(users, u)
	}

	return users, nil
}

func (r *UserRepository) Availability(ctx context.Context, email *string, username *string) (bool, bool, error) {
	var emailStr, usernameStr string
	if email != nil {
		emailStr = *email
	}
	if username != nil {
		usernameStr = *username
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
		ID:                     db.ToUUID(u.ID),
		Email:                  u.Email,
		Username:               u.Username,
		DisplayName:            u.DisplayName,
		PasswordHash:           u.PasswordHash,
		Phone:                  db.ToTextPtr(u.Phone),
		Bio:                    db.ToTextPtr(u.Bio),
		AvatarURL:              db.ToTextPtr(u.AvatarURL),
		BannerColor:            db.ToTextPtr(u.BannerColor),
		PreferredPresence:      db.ToInt2Ptr(u.PreferredPresence),
		PreferredPresenceUntil: db.ToTimestamptzPtr(u.PreferredPresenceUntil),
		VerifiedAt:             db.ToTimestamptzPtr(u.VerifiedAt),
		DisabledAt:             db.ToTimestamptzPtr(u.DisabledAt),
		DeleteScheduledAt:      db.ToTimestamptzPtr(u.DeleteScheduledAt),
		UpdatedAt:              db.ToTimestamptz(u.UpdatedAt),
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return userFromRow(row), nil
}

func (r *UserRepository) UpdateEmail(ctx context.Context, id uuid.UUID, email string, updatedAt time.Time) (*user.User, error) {
	row, err := r.store.UserUpdateEmail(ctx, db.UserUpdateEmailParams{
		ID:        db.ToUUID(id),
		Email:     email,
		UpdatedAt: db.ToTimestamptz(updatedAt),
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return userFromRow(row), nil
}

func (r *UserRepository) UpdateUsername(ctx context.Context, id uuid.UUID, username string, updatedAt time.Time) (*user.User, error) {
	row, err := r.store.UserUpdateUsername(ctx, db.UserUpdateUsernameParams{
		ID:        db.ToUUID(id),
		Username:  username,
		UpdatedAt: db.ToTimestamptz(updatedAt),
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return userFromRow(row), nil
}

func (r *UserRepository) UpdatePhone(ctx context.Context, id uuid.UUID, phone *string, updatedAt time.Time) (*user.User, error) {
	row, err := r.store.UserUpdatePhone(ctx, db.UserUpdatePhoneParams{
		ID:        db.ToUUID(id),
		Phone:     db.ToTextPtr(phone),
		UpdatedAt: db.ToTimestamptz(updatedAt),
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return userFromRow(row), nil
}

func (r *UserRepository) UpdatePasswordHash(ctx context.Context, id uuid.UUID, passwordHash string, updatedAt time.Time) (*user.User, error) {
	row, err := r.store.UserUpdatePasswordHash(ctx, db.UserUpdatePasswordHashParams{
		ID:           db.ToUUID(id),
		PasswordHash: passwordHash,
		UpdatedAt:    db.ToTimestamptz(updatedAt),
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return userFromRow(row), nil
}

func (r *UserRepository) UpdateProfile(
	ctx context.Context,
	id uuid.UUID,
	displayName string,
	bio *string,
	avatarURL *string,
	bannerColor *string,
	updatedAt time.Time,
) (*user.User, error) {
	row, err := r.store.UserUpdateProfile(ctx, db.UserUpdateProfileParams{
		ID:          db.ToUUID(id),
		DisplayName: displayName,
		Bio:         db.ToTextPtr(bio),
		AvatarURL:   db.ToTextPtr(avatarURL),
		BannerColor: db.ToTextPtr(bannerColor),
		UpdatedAt:   db.ToTimestamptz(updatedAt),
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return userFromRow(row), nil
}

func (r *UserRepository) UpdatePresence(
	ctx context.Context,
	id uuid.UUID,
	presence *presence.Presence,
	presenceUntil *time.Time,
	updatedAt time.Time,
) (*user.User, error) {
	row, err := r.store.UserUpdatePresence(ctx, db.UserUpdatePresenceParams{
		ID:                     db.ToUUID(id),
		PreferredPresence:      db.ToInt2Ptr(presence),
		PreferredPresenceUntil: db.ToTimestamptzPtr(presenceUntil),
		UpdatedAt:              db.ToTimestamptz(updatedAt),
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return userFromRow(row), nil
}

func (r *UserRepository) Verify(ctx context.Context, id uuid.UUID, verifiedAt *time.Time, updatedAt time.Time) (*user.User, error) {
	row, err := r.store.UserVerify(ctx, db.UserVerifyParams{
		ID:         db.ToUUID(id),
		VerifiedAt: db.ToTimestamptzPtr(verifiedAt),
		UpdatedAt:  db.ToTimestamptz(updatedAt),
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return userFromRow(row), nil
}

func (r *UserRepository) SetDisabled(ctx context.Context, id uuid.UUID, disabledAt *time.Time, updatedAt time.Time) (*user.User, error) {
	row, err := r.store.UserSetDisabled(ctx, db.UserSetDisabledParams{
		ID:         db.ToUUID(id),
		DisabledAt: db.ToTimestamptzPtr(disabledAt),
		UpdatedAt:  db.ToTimestamptz(updatedAt),
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return userFromRow(row), nil
}

func (r *UserRepository) SetDeleteSchedule(ctx context.Context, id uuid.UUID, deleteScheduledAt, disabledAt *time.Time, updatedAt time.Time) (*user.User, error) {
	row, err := r.store.UserSetDeleteSchedule(ctx, db.UserSetDeleteScheduleParams{
		ID:                db.ToUUID(id),
		DeleteScheduledAt: db.ToTimestamptzPtr(deleteScheduledAt),
		DisabledAt:        db.ToTimestamptzPtr(disabledAt),
		UpdatedAt:         db.ToTimestamptz(updatedAt),
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return userFromRow(row), nil
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
		var preferredPresence *int16
		if u.PreferredPresence != nil {
			val := int16(*u.PreferredPresence)
			preferredPresence = &val
		}

		payloads[i] = userPayload{
			ID:                     u.ID,
			Email:                  u.Email,
			Username:               u.Username,
			DisplayName:            u.DisplayName,
			PasswordHash:           u.PasswordHash,
			Phone:                  u.Phone,
			Bio:                    u.Bio,
			AvatarURL:              u.AvatarURL,
			BannerColor:            u.BannerColor,
			PreferredPresence:      preferredPresence,
			PreferredPresenceUntil: u.PreferredPresenceUntil,
			VerifiedAt:             u.VerifiedAt,
			DisabledAt:             u.DisabledAt,
			DeleteScheduledAt:      u.DeleteScheduledAt,
			CreatedAt:              u.CreatedAt,
			UpdatedAt:              u.UpdatedAt,
		}
	}

	jsonBytes, err := json.Marshal(payloads)
	if err != nil {
		return nil, errs.Internal("failed to marshal user update batch payload").
			Meta("scope", "user").
			Wrap(err)
	}

	rows, err := r.store.UserUpdateBatch(ctx, jsonBytes)
	if err != nil {
		return nil, r.store.Err(err)
	}

	updatedUsers := make([]*user.User, len(rows))
	for i, row := range rows {
		u := userFromRow(row)
		updatedUsers[i] = u
	}

	return updatedUsers, nil
}

func userFromRow(row db.User) *user.User {
	var preferredPresence *presence.Presence
	if p, err := presence.Parse(db.FromInt2[int](row.PreferredPresence)); err == nil {
		preferredPresence = &p
	}

	return user.Reconstitute(
		db.FromUUID[uuid.UUID](row.ID),
		row.Email,
		row.Username,
		row.PasswordHash,
		db.FromTextPtr[string](row.Phone),
		row.DisplayName,
		db.FromTextPtr[string](row.Bio),
		db.FromTextPtr[string](row.AvatarURL),
		db.FromTextPtr[string](row.BannerColor),
		preferredPresence,
		db.FromTimestamptzPtr(row.PreferredPresenceUntil),
		db.FromTimestamptzPtr(row.VerifiedAt),
		db.FromTimestamptzPtr(row.DisabledAt),
		db.FromTimestamptzPtr(row.DeleteScheduledAt),
		db.FromTimestamptz(row.CreatedAt),
		db.FromTimestamptz(row.UpdatedAt),
	)
}
