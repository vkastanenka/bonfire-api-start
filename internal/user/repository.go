package user

import (
	"bonfire-api/internal/repository"
	"bonfire-api/internal/store"
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type Repository struct {
	store store.Store // Or whatever your raw query interface is named
}

func NewRepository(store store.Store) *Repository {
	return &Repository{store: store}
}

type CheckAvailabilityParams struct {
	Email    string `json:"email"`
	Username string `json:"username"`
}

type CheckAvailabilityResult struct {
	Email    bool `json:"email"`
	Username bool `json:"username"`
}

func (r *Repository) CheckAvailability(ctx context.Context, p CheckAvailabilityParams) (CheckAvailabilityResult, error) {
	row, err := r.store.UserCheckAvailability(ctx, store.UserCheckAvailabilityParams{
		Email:    p.Email,
		Username: p.Username,
	})
	if err != nil {
		return CheckAvailabilityResult{Email: false, Username: false}, repository.NewError(err, repository.ScopeUser)
	}
	return CheckAvailabilityResult{Email: row.EmailAvailable, Username: row.UsernameAvailable}, nil
}

func (r *Repository) Create(ctx context.Context, id uuid.UUID, email, username, hash string) (User, error) {
	row, err := r.store.UserCreate(ctx, store.UserCreateParams{
		ID:           pgtype.UUID{Bytes: id, Valid: true},
		Email:        email,
		Username:     username,
		PasswordHash: hash,
	})
	if err != nil {
		return User{}, repository.NewError(err, repository.ScopeUser)
	}
	return FromRepository(row), nil
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (User, error) {
	row, err := r.store.UserGetByID(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return User{}, repository.NewError(err, repository.ScopeUser)
	}
	return FromRepository(row), nil
}

func (r *Repository) GetByEmail(ctx context.Context, email string) (User, error) {
	row, err := r.store.UserGetByEmail(ctx, email)
	if err != nil {
		return User{}, repository.NewError(err, repository.ScopeUser)
	}
	return FromRepository(row), nil
}

func (r *Repository) GetByUsername(ctx context.Context, username string) (User, error) {
	row, err := r.store.UserGetByUsername(ctx, username)
	if err != nil {
		return User{}, repository.NewError(err, repository.ScopeUser)
	}
	return FromRepository(row), nil
}

func (r *Repository) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) (User, error) {
	row, err := r.store.UserUpdatePassword(ctx, repository.UserUpdatePasswordParams{
		ID:           pgtype.UUID{Bytes: id, Valid: true},
		PasswordHash: passwordHash,
	})
	if err != nil {
		return User{}, repository.NewError(err, repository.ScopeUser)
	}
	return FromRepository(row), nil
}

func (r *Repository) MarkVerified(ctx context.Context, id uuid.UUID) (User, error) {
	row, err := s.store.UserMarkVerified(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return User{}, repository.NewError(err, repository.ScopeUser)
	}
	return FromRepository(row), nil
}

type CreateProfileParams struct {
	UserID      uuid.UUID
	DisplayName string
}

func (r *Repository) CreateProfile(ctx context.Context, p CreateProfileParams) (UserProfile, error) {
	row, err := s.store.UserProfileCreate(ctx, repository.UserProfileCreateParams{
		UserID:      pgtype.UUID{Bytes: p.UserID, Valid: true},
		DisplayName: p.DisplayName,
	})
	if err != nil {
		return UserProfile{}, repository.NewError(err, repository.ScopeUserProfile)
	}
	return ProfileFromRepository(row), nil
}

func (r *Repository) GetProfileByUserID(ctx context.Context, id uuid.UUID) (UserProfile, error) {
	row, err := s.store.UserProfileGetByUserID(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return UserProfile{}, repository.NewError(err, repository.ScopeUserProfile)
	}
	return ProfileFromRepository(row), nil
}
