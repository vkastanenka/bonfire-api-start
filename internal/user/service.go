package user

import (
	"bonfire-api/internal/repository"
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// Store defines the database actions owned by the User domain.
// Generated sqlc methods fit here perfectly.
type Store interface {
	GetUserByID(ctx context.Context, id pgtype.UUID) (repository.GetUserByIDRow, error)
	GetUserByEmail(ctx context.Context, email string) (repository.GetUserByEmailRow, error)
	CreateUserProfile(ctx context.Context, arg repository.CreateUserProfileParams) (repository.CreateUserProfileRow, error)
	ValidateUserCredentialsAvailability(ctx context.Context, arg repository.ValidateUserCredentialsAvailabilityParams) (repository.ValidateUserCredentialsAvailabilityRow, error)
}

type UserService struct {
	store Store
}

func NewUserService(store Store) *UserService {
	return &UserService{store: store}
}

// GetUserByID safely transforms the incoming application UUID into the
// database-friendly pgtype.UUID and fetches the record.
func (s *UserService) GetUserByID(ctx context.Context, userID uuid.UUID) (repository.GetUserByIDRow, error) {
	var pgUserID pgtype.UUID
	pgUserID.Bytes = userID
	pgUserID.Valid = true

	return s.store.GetUserByID(ctx, pgUserID)
}

// GetUserByEmail searches for a user record using their unique email address.
func (s *UserService) GetUserByEmail(ctx context.Context, email string) (repository.GetUserByEmailRow, error) {
	return s.store.GetUserByEmail(ctx, email)
}
