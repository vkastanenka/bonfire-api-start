package repository

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/pkg/ptr"
	"bonfire-api/internal/presence" // TODO: Remove dependency
	"bonfire-api/internal/store"
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type User struct {
	ID           uuid.UUID
	Email        string
	Username     string
	PasswordHash string
	Presence     *presence.Presence // TODO: Refactor
	VerifiedAt   *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (u User) IsVerified() bool {
	return u.VerifiedAt != nil
}

func FromRepository(row store.User) User {
	u := User{
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
		u.Presence = ptr.To(presence.Presence(row.Presence.Int16))
	}

	return u
}

type UserProfile struct {
	UserID      uuid.UUID
	DisplayName string
	AvatarURL   *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func ProfileFromRepository(row store.UserProfile) UserProfile {
	up := UserProfile{
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

type UserRepository struct {
	store store.Store
}

func NewUserRepository(store store.Store) *UserRepository {
	return &UserRepository{store: store}
}

type CheckAvailabilityParams struct {
	Email    string `json:"email"`
	Username string `json:"username"`
}

type CheckAvailabilityResult struct {
	Email    bool `json:"email"`
	Username bool `json:"username"`
}

func (r *UserRepository) CheckAvailability(ctx context.Context, p CheckAvailabilityParams) (CheckAvailabilityResult, error) {
	row, err := r.store.UserCheckAvailability(ctx, store.UserCheckAvailabilityParams{
		Email:    p.Email,
		Username: p.Username,
	})
	if err != nil {
		return CheckAvailabilityResult{Email: false, Username: false}, NewError(err, ScopeUser)
	}
	return CheckAvailabilityResult{Email: row.EmailAvailable, Username: row.UsernameAvailable}, nil
}

type CreateParams struct {
	ID       *uuid.UUID `json:"id,omitempty"`
	Email    string     `json:"email"`
	Username string     `json:"username"`
	Password string     `json:"password"`
}

func (r *UserRepository) Create(ctx context.Context, p CreateParams) (User, error) {
	var targetID uuid.UUID

	if p.ID != nil {
		targetID = *p.ID
	} else {
		var err error
		targetID, err = uuid.NewV7()
		if err != nil {
			return User{}, apperr.NewInternal(err, "", "")
		}
	}

	row, err := r.store.UserCreate(ctx, store.UserCreateParams{
		ID:           pgtype.UUID{Bytes: targetID, Valid: true},
		Email:        p.Email,
		Username:     p.Username,
		PasswordHash: p.Password,
	})
	if err != nil {
		return User{}, NewError(err, ScopeUser)
	}
	return FromRepository(row), nil
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (User, error) {
	row, err := r.store.UserGetByID(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return User{}, NewError(err, ScopeUser)
	}
	return FromRepository(row), nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (User, error) {
	row, err := r.store.UserGetByEmail(ctx, email)
	if err != nil {
		return User{}, NewError(err, ScopeUser)
	}
	return FromRepository(row), nil
}

func (r *UserRepository) GetByUsername(ctx context.Context, username string) (User, error) {
	row, err := r.store.UserGetByUsername(ctx, username)
	if err != nil {
		return User{}, NewError(err, ScopeUser)
	}
	return FromRepository(row), nil
}

func (r *UserRepository) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) (User, error) {
	row, err := r.store.UserUpdatePassword(ctx, store.UserUpdatePasswordParams{
		ID:           pgtype.UUID{Bytes: id, Valid: true},
		PasswordHash: passwordHash,
	})
	if err != nil {
		return User{}, NewError(err, ScopeUser)
	}
	return FromRepository(row), nil
}

func (r *UserRepository) MarkVerified(ctx context.Context, id uuid.UUID) (User, error) {
	row, err := r.store.UserMarkVerified(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return User{}, NewError(err, ScopeUser)
	}
	return FromRepository(row), nil
}

type CreateProfileParams struct {
	UserID      uuid.UUID
	DisplayName string
}

func (r *UserRepository) CreateProfile(ctx context.Context, p CreateProfileParams) (UserProfile, error) {
	row, err := r.store.UserProfileCreate(ctx, store.UserProfileCreateParams{
		UserID:      pgtype.UUID{Bytes: p.UserID, Valid: true},
		DisplayName: p.DisplayName,
	})
	if err != nil {
		return UserProfile{}, NewError(err, ScopeUserProfile)
	}
	return ProfileFromRepository(row), nil
}

func (s *UserRepository) GetProfileByUserID(ctx context.Context, id uuid.UUID) (UserProfile, error) {
	row, err := s.store.UserProfileGetByUserID(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return UserProfile{}, NewError(err, ScopeUserProfile)
	}
	return ProfileFromRepository(row), nil
}
