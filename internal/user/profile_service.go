package user

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/repository"
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// ==========================================
// META
// ==========================================

func (s *Service) CountProfiles(ctx context.Context) (int64, error) {
	count, err := s.store.UserProfileCount(ctx)
	if err != nil {
		return 0, apperr.NewDBError(err, DomainProfile)
	}
	return count, nil
}

// ==========================================
// CREATE
// ==========================================

// Create
func (s *Service) CreateProfile(ctx context.Context, p CreateProfileParams) (ProfileView, error) {
	row, err := s.store.UserProfileCreate(ctx, repository.UserProfileCreateParams{
		UserID:      pgtype.UUID{Bytes: p.UserID, Valid: true},
		DisplayName: p.DisplayName,
	})
	if err != nil {
		return ProfileView{}, apperr.NewDBError(err, Domain)
	}
	return NewProfileView(row), nil
}

// ==========================================
// GET
// ==========================================

// GetProfileByUserID
func (s *Service) GetProfileByUserID(ctx context.Context, userID uuid.UUID) (ProfileView, error) {
	row, err := s.store.UserProfileGetByUserID(ctx, pgtype.UUID{Bytes: userID, Valid: true})
	if err != nil {
		return ProfileView{}, apperr.NewDBError(err, DomainProfile)
	}
	return NewProfileView(row), nil
}

// ==========================================
// UPDATE
// ==========================================

// UpdateProfileDisplayName
func (s *Service) UpdateProfileDisplayName(ctx context.Context, p UpdateProfileDisplayNameParams) (ProfileView, error) {
	row, err := s.store.UserProfileUpdateDisplayName(ctx, repository.UserProfileUpdateDisplayNameParams{
		UserID:      pgtype.UUID{Bytes: p.UserID, Valid: true},
		DisplayName: p.DisplayName,
	})
	if err != nil {
		return ProfileView{}, apperr.NewDBError(err, DomainProfile)
	}
	return NewProfileView(row), nil
}

// ==========================================
// DELETE
// ==========================================

// DeleteProfileByUserID
func (s *Service) DeleteProfileByUserID(ctx context.Context, userID uuid.UUID) error {
	err := s.store.UserProfileDeleteByUserID(ctx, pgtype.UUID{Bytes: userID, Valid: true})
	if err != nil {
		return apperr.NewDBError(err, DomainProfile)
	}
	return nil
}
