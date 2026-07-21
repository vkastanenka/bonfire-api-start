package repository

import (
	"context"

	"bonfire-api/internal/db"
	"bonfire-api/internal/pkg/ptr"
	"bonfire-api/internal/user"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type User struct {
	store db.Store
}

func NewUser(store db.Store) *User {
	return &User{store: store}
}

func (r *User) CheckAvailability(ctx context.Context, p user.CheckAvailabilityParams) (user.CheckAvailabilityResult, error) {
	row, err := r.store.UserCheckAvailability(ctx, db.UserCheckAvailabilityParams{
		Email:    p.Email,
		Username: p.Username,
	})
	if err != nil {
		return user.CheckAvailabilityResult{}, NewError(err, EntityUser)
	}

	return user.CheckAvailabilityResult{
		EmailAvailable:    row.EmailAvailable,
		UsernameAvailable: row.UsernameAvailable,
	}, nil
}

func (r *User) Create(ctx context.Context, p user.CreateParams) (user.User, error) {
	row, err := r.store.UserCreate(ctx, db.UserCreateParams{
		ID:           pgtype.UUID{Bytes: p.ID, Valid: p.ID != uuid.Nil},
		Email:        p.Email,
		Username:     p.Username,
		PasswordHash: p.PasswordHash,
	})
	if err != nil {
		return user.User{}, NewError(err, EntityUser)
	}

	return userFromDB(row), nil
}

func (r *User) GetByID(ctx context.Context, id uuid.UUID) (user.User, error) {
	row, err := r.store.UserGetByID(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return user.User{}, NewError(err, EntityUser)
	}
	return userFromDB(row), nil
}

func (r *User) GetByEmail(ctx context.Context, email string) (user.User, error) {
	row, err := r.store.UserGetByEmail(ctx, email)
	if err != nil {
		return user.User{}, NewError(err, EntityUser)
	}
	return userFromDB(row), nil
}

func (r *User) GetByUsername(ctx context.Context, username string) (user.User, error) {
	row, err := r.store.UserGetByUsername(ctx, username)
	if err != nil {
		return user.User{}, NewError(err, EntityUser)
	}
	return userFromDB(row), nil
}

func (r *User) UpdatePassword(ctx context.Context, p user.UpdatePasswordParams) (user.User, error) {
	row, err := r.store.UserUpdatePassword(ctx, db.UserUpdatePasswordParams{
		ID:           pgtype.UUID{Bytes: p.ID, Valid: true},
		PasswordHash: p.PasswordHash,
	})
	if err != nil {
		return user.User{}, NewError(err, EntityUser)
	}
	return userFromDB(row), nil
}

func (r *User) MarkVerified(ctx context.Context, id uuid.UUID) (user.User, error) {
	row, err := r.store.UserMarkVerified(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return user.User{}, NewError(err, EntityUser)
	}
	return userFromDB(row), nil
}

func (r *User) CreateProfile(ctx context.Context, p user.CreateProfileParams) (user.UserProfile, error) {
	row, err := r.store.UserProfileCreate(ctx, db.UserProfileCreateParams{
		UserID:      pgtype.UUID{Bytes: p.UserID, Valid: true},
		DisplayName: p.DisplayName,
	})
	if err != nil {
		return user.UserProfile{}, NewError(err, EntityUserProfile)
	}
	return userProfileFromDB(row), nil
}

func (r *User) GetProfileByUserID(ctx context.Context, userID uuid.UUID) (user.UserProfile, error) {
	row, err := r.store.UserProfileGetByUserID(ctx, pgtype.UUID{Bytes: userID, Valid: true})
	if err != nil {
		return user.UserProfile{}, NewError(err, EntityUserProfile)
	}
	return userProfileFromDB(row), nil
}

func userFromDB(row db.User) user.User {
	u := user.User{
		ID:           uuid.UUID(row.ID.Bytes),
		Email:        row.Email,
		Username:     row.Username,
		PasswordHash: row.PasswordHash,
		CreatedAt:    row.CreatedAt.Time,
		UpdatedAt:    row.UpdatedAt.Time,
	}

	if row.VerifiedAt.Valid {
		u.VerifiedAt = ptr.To(row.VerifiedAt.Time)
	}

	if row.Presence.Valid {
		u.Presence = user.Presence(row.Presence.Int16)
	}

	return u
}

func userProfileFromDB(row db.UserProfile) user.UserProfile {
	up := user.UserProfile{
		UserID:      uuid.UUID(row.UserID.Bytes),
		DisplayName: row.DisplayName,
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
	}

	if row.AvatarUrl.Valid {
		up.AvatarURL = ptr.To(row.AvatarUrl.String)
	}

	return up
}

var _ user.Repository = (*User)(nil)
