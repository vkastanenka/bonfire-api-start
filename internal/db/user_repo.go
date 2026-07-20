package db

import (
	// TODO: Remove dependency
	"bonfire-api/internal/pkg/ptr"
	"bonfire-api/internal/presence"
	"bonfire-api/internal/user"
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type UserRepository struct {
	store Store
}

func NewUserRepository(store Store) *UserRepository {
	return &UserRepository{store: store}
}

func (r *UserRepository) CheckAvailability(ctx context.Context, email, username string) (bool, bool, error) {
	row, err := r.store.UserCheckAvailability(ctx, UserCheckAvailabilityParams{
		Email:    email,
		Username: username,
	})
	if err != nil {
		return false, false, NewError(err, EntityUser)
	}
	return row.EmailAvailable, row.UsernameAvailable, nil
}

func (r *UserRepository) Create(ctx context.Context, id uuid.UUID, email, username, password string) (user.User, error) {
	row, err := r.store.UserCreate(ctx, UserCreateParams{
		ID:           pgtype.UUID{Bytes: id, Valid: true},
		Email:        email,
		Username:     username,
		PasswordHash: password,
	})
	if err != nil {
		return user.User{}, NewError(err, EntityUser)
	}
	return userFromDB(row), nil
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (user.User, error) {
	row, err := r.store.UserGetByID(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return user.User{}, NewError(err, EntityUser)
	}
	return userFromDB(row), nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (user.User, error) {
	row, err := r.store.UserGetByEmail(ctx, email)
	if err != nil {
		return user.User{}, NewError(err, EntityUser)
	}
	return userFromDB(row), nil
}

func (r *UserRepository) GetByUsername(ctx context.Context, username string) (user.User, error) {
	row, err := r.store.UserGetByUsername(ctx, username)
	if err != nil {
		return user.User{}, NewError(err, EntityUser)
	}
	return userFromDB(row), nil
}

func (r *UserRepository) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) (user.User, error) {
	row, err := r.store.UserUpdatePassword(ctx, UserUpdatePasswordParams{
		ID:           pgtype.UUID{Bytes: id, Valid: true},
		PasswordHash: passwordHash,
	})
	if err != nil {
		return user.User{}, NewError(err, EntityUser)
	}
	return userFromDB(row), nil
}

func (r *UserRepository) MarkVerified(ctx context.Context, id uuid.UUID) (user.User, error) {
	row, err := r.store.UserMarkVerified(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return user.User{}, NewError(err, EntityUser)
	}
	return userFromDB(row), nil
}

func (r *UserRepository) CreateProfile(ctx context.Context, userID uuid.UUID, displayName string) (user.UserProfile, error) {
	row, err := r.store.UserProfileCreate(ctx, UserProfileCreateParams{
		UserID:      pgtype.UUID{Bytes: userID, Valid: true},
		DisplayName: displayName,
	})
	if err != nil {
		return user.UserProfile{}, NewError(err, EntityUserProfile)
	}
	return userProfileFromDB(row), nil
}

func (s *UserRepository) GetProfileByUserID(ctx context.Context, id uuid.UUID) (user.UserProfile, error) {
	row, err := s.store.UserProfileGetByUserID(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return user.UserProfile{}, NewError(err, EntityUserProfile)
	}
	return userProfileFromDB(row), nil
}

func userFromDB(row User) user.User {
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
		p := presence.Presence(row.Presence.Int16)
		u.Presence = &p
	}

	return u
}

func userProfileFromDB(row UserProfile) user.UserProfile {
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
