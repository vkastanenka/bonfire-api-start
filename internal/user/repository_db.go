package user

import (
	"context"
	"encoding/json"
	"time"

	"bonfire-api/internal/db"
	"bonfire-api/internal/gateway"

	"github.com/google/uuid"
)

type Repository struct {
	store *db.Store
}

func NewRepository(store *db.Store) *Repository {
	return &Repository{store: store}
}

func (r *Repository) Create(ctx context.Context, u *User) (*User, error) {
	if u == nil {
		return nil, ErrPayloadRequired()
	}

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
		return nil, db.NewError(err, db.ResourceUser)
	}

	return userFromRow(row), nil
}

func (r *Repository) Get(ctx context.Context, id uuid.UUID) (*User, error) {
	row, err := r.store.UserGet(ctx, db.ToUUID(id))
	if err != nil {
		return nil, db.NewError(err, db.ResourceUser)
	}

	return userFromRow(row), nil
}

func (r *Repository) GetByEmail(ctx context.Context, email string) (*User, error) {
	row, err := r.store.UserGetByEmail(ctx, email)
	if err != nil {
		return nil, db.NewError(err, db.ResourceUser)
	}

	return userFromRow(row), nil
}

func (r *Repository) GetByUsername(ctx context.Context, username string) (*User, error) {
	row, err := r.store.UserGetByUsername(ctx, username)
	if err != nil {
		return nil, db.NewError(err, db.ResourceUser)
	}

	return userFromRow(row), nil
}

func (r *Repository) GetBatch(ctx context.Context, ids []uuid.UUID) ([]*User, error) {
	rows, err := r.store.UserGetBatch(ctx, db.ToUUIDs(ids))
	if err != nil {
		return nil, db.NewError(err, db.ResourceUser)
	}

	return usersFromRows(rows), nil
}

func (r *Repository) ListDeleteScheduled(ctx context.Context, currentTime time.Time, limitVal int) ([]*User, error) {
	rows, err := r.store.UserListDeleteScheduled(ctx, db.UserListDeleteScheduledParams{
		Now:      db.ToTimestamptz(currentTime),
		LimitVal: int32(limitVal),
	})
	if err != nil {
		return nil, db.NewError(err, db.ResourceUser)
	}

	return usersFromRows(rows), nil
}

func (r *Repository) Update(ctx context.Context, u *User) (*User, error) {
	if u == nil {
		return nil, ErrPayloadRequired()
	}

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
		return nil, db.NewError(err, db.ResourceUser)
	}

	return userFromRow(row), nil
}

func (r *Repository) UpdateEmail(ctx context.Context, id uuid.UUID, email string, updatedAt time.Time) (*User, error) {
	row, err := r.store.UserUpdateEmail(ctx, db.UserUpdateEmailParams{
		ID:        db.ToUUID(id),
		Email:     email,
		UpdatedAt: db.ToTimestamptz(updatedAt),
	})
	if err != nil {
		return nil, db.NewError(err, db.ResourceUser)
	}

	return userFromRow(row), nil
}

func (r *Repository) UpdateUsername(ctx context.Context, id uuid.UUID, username string, updatedAt time.Time) (*User, error) {
	row, err := r.store.UserUpdateUsername(ctx, db.UserUpdateUsernameParams{
		ID:        db.ToUUID(id),
		Username:  username,
		UpdatedAt: db.ToTimestamptz(updatedAt),
	})
	if err != nil {
		return nil, db.NewError(err, db.ResourceUser)
	}

	return userFromRow(row), nil
}

func (r *Repository) UpdatePhone(ctx context.Context, id uuid.UUID, phone *string, updatedAt time.Time) (*User, error) {
	row, err := r.store.UserUpdatePhone(ctx, db.UserUpdatePhoneParams{
		ID:        db.ToUUID(id),
		Phone:     db.ToTextPtr(phone),
		UpdatedAt: db.ToTimestamptz(updatedAt),
	})
	if err != nil {
		return nil, db.NewError(err, db.ResourceUser)
	}

	return userFromRow(row), nil
}

func (r *Repository) UpdatePasswordHash(ctx context.Context, id uuid.UUID, passwordHash string, updatedAt time.Time) (*User, error) {
	row, err := r.store.UserUpdatePasswordHash(ctx, db.UserUpdatePasswordHashParams{
		ID:           db.ToUUID(id),
		PasswordHash: passwordHash,
		UpdatedAt:    db.ToTimestamptz(updatedAt),
	})
	if err != nil {
		return nil, db.NewError(err, db.ResourceUser)
	}

	return userFromRow(row), nil
}

func (r *Repository) UpdateProfile(
	ctx context.Context,
	id uuid.UUID,
	displayName string,
	bio *string,
	avatarURL *string,
	bannerColor *string,
	updatedAt time.Time,
) (*User, error) {
	row, err := r.store.UserUpdateProfile(ctx, db.UserUpdateProfileParams{
		ID:          db.ToUUID(id),
		DisplayName: displayName,
		Bio:         db.ToTextPtr(bio),
		AvatarURL:   db.ToTextPtr(avatarURL),
		BannerColor: db.ToTextPtr(bannerColor),
		UpdatedAt:   db.ToTimestamptz(updatedAt),
	})
	if err != nil {
		return nil, db.NewError(err, db.ResourceUser)
	}

	return userFromRow(row), nil
}

func (r *Repository) UpdatePreferredPresence(
	ctx context.Context,
	id uuid.UUID,
	preferredPresence *gateway.Presence,
	preferredPresenceUntil *time.Time,
	updatedAt time.Time,
) (*User, error) {
	row, err := r.store.UserUpdatePresence(ctx, db.UserUpdatePresenceParams{
		ID:                     db.ToUUID(id),
		PreferredPresence:      db.ToInt2Ptr(preferredPresence),
		PreferredPresenceUntil: db.ToTimestamptzPtr(preferredPresenceUntil),
		UpdatedAt:              db.ToTimestamptz(updatedAt),
	})
	if err != nil {
		return nil, db.NewError(err, db.ResourceUser)
	}

	return userFromRow(row), nil
}

func (r *Repository) Verify(ctx context.Context, id uuid.UUID, verifiedAt *time.Time, updatedAt time.Time) (*User, error) {
	row, err := r.store.UserVerify(ctx, db.UserVerifyParams{
		ID:         db.ToUUID(id),
		VerifiedAt: db.ToTimestamptzPtr(verifiedAt),
		UpdatedAt:  db.ToTimestamptz(updatedAt),
	})
	if err != nil {
		return nil, db.NewError(err, db.ResourceUser)
	}

	return userFromRow(row), nil
}

func (r *Repository) SetDisabled(ctx context.Context, id uuid.UUID, disabledAt *time.Time, updatedAt time.Time) (*User, error) {
	row, err := r.store.UserSetDisabled(ctx, db.UserSetDisabledParams{
		ID:         db.ToUUID(id),
		DisabledAt: db.ToTimestamptzPtr(disabledAt),
		UpdatedAt:  db.ToTimestamptz(updatedAt),
	})
	if err != nil {
		return nil, db.NewError(err, db.ResourceUser)
	}

	return userFromRow(row), nil
}

func (r *Repository) SetDeleteSchedule(ctx context.Context, id uuid.UUID, deleteScheduledAt, disabledAt *time.Time, updatedAt time.Time) (*User, error) {
	row, err := r.store.UserSetDeleteSchedule(ctx, db.UserSetDeleteScheduleParams{
		ID:                db.ToUUID(id),
		DeleteScheduledAt: db.ToTimestamptzPtr(deleteScheduledAt),
		DisabledAt:        db.ToTimestamptzPtr(disabledAt),
		UpdatedAt:         db.ToTimestamptz(updatedAt),
	})
	if err != nil {
		return nil, db.NewError(err, db.ResourceUser)
	}

	return userFromRow(row), nil
}

func (r *Repository) UpdateBatch(ctx context.Context, users []*User) ([]*User, error) {
	if len(users) == 0 {
		return nil, nil
	}

	jsonBytes, err := json.Marshal(newUserUpdatePayloads(users))
	if err != nil {
		return nil, ErrJSONMarshalFailed().Wrap(err)
	}

	rows, err := r.store.UserUpdateBatch(ctx, jsonBytes)
	if err != nil {
		return nil, db.NewError(err, db.ResourceUser)
	}

	return usersFromRows(rows), nil
}

func userFromRow(row db.User) *User {
	var preferredPresence *gateway.Presence
	if p, err := gateway.ParsePresenceInt(db.FromInt2(row.PreferredPresence)); err == nil {
		preferredPresence = &p
	}

	return Reconstitute(
		db.FromUUID(row.ID),
		row.Email,
		row.Username,
		row.PasswordHash,
		db.FromTextPtr(row.Phone),
		row.DisplayName,
		db.FromTextPtr(row.Bio),
		db.FromTextPtr(row.AvatarURL),
		db.FromTextPtr(row.BannerColor),
		preferredPresence,
		db.FromTimestamptzPtr(row.PreferredPresenceUntil),
		db.FromTimestamptzPtr(row.VerifiedAt),
		db.FromTimestamptzPtr(row.DisabledAt),
		db.FromTimestamptzPtr(row.DeleteScheduledAt),
		db.FromTimestamptz(row.CreatedAt),
		db.FromTimestamptz(row.UpdatedAt),
	)
}

func usersFromRows(rows []db.User) []*User {
	if len(rows) == 0 {
		return nil
	}

	users := make([]*User, 0, len(rows))
	for _, row := range rows {
		users = append(users, userFromRow(row))
	}

	return users
}

type userUpdatePayload struct {
	ID                     uuid.UUID  `json:"id"`
	Email                  string     `json:"email"`
	Username               string     `json:"username"`
	DisplayName            string     `json:"display_name"`
	PasswordHash           string     `json:"password_hash"`
	Phone                  *string    `json:"phone"`
	Bio                    *string    `json:"bio"`
	AvatarURL              *string    `json:"avatar_url"`
	BannerColor            *string    `json:"banner_color"`
	PreferredPresence      *int16     `json:"preferred_presence"`
	PreferredPresenceUntil *time.Time `json:"preferred_presence_until"`
	VerifiedAt             *time.Time `json:"verified_at"`
	DisabledAt             *time.Time `json:"disabled_at"`
	DeleteScheduledAt      *time.Time `json:"delete_scheduled_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
}

func newUserUpdatePayload(u *User) *userUpdatePayload {
	if u == nil {
		return nil
	}

	var preferredPresence *int16 = nil
	if u.PreferredPresence != nil {
		v := int16(*u.PreferredPresence)
		preferredPresence = &v
	}

	return &userUpdatePayload{
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
		UpdatedAt:              u.UpdatedAt,
	}
}

func newUserUpdatePayloads(users []*User) []*userUpdatePayload {
	payloads := make([]*userUpdatePayload, 0, len(users))
	for _, u := range users {
		if p := newUserUpdatePayload(u); p != nil {
			payloads = append(payloads, p)
		}
	}
	return payloads
}
