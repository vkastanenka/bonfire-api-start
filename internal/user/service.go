package user

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/repository"
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type Store interface {
	repository.Querier
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

// ==========================================
// META
// ==========================================

// Count
func (s *Service) Count(ctx context.Context) (int64, error) {
	count, err := s.store.OutboxEventCount(ctx)
	if err != nil {
		return 0, apperr.NewDBError(err, User)
	}
	return count, nil
}

func (s *Service) CheckAvailability(ctx context.Context, p CheckAvailabilityParams) (CheckAvailabilityResult, error) {
	row, err := s.store.UserCheckAvailability(ctx, repository.UserCheckAvailabilityParams{
		Email:    p.Email,
		Username: p.Username,
	})
	if err != nil {
		return CheckAvailabilityResult{Email: false, Username: false}, apperr.NewDBError(err, User)
	}
	return CheckAvailabilityResult{Email: row.EmailAvailable, Username: row.UsernameAvailable}, nil
}

// ==========================================
// CREATE
// ==========================================

func (s *Service) Create(ctx context.Context, p CreateParams) (View, error) {
	row, err := s.store.UserCreate(ctx, repository.UserCreateParams{
		Email:        p.Email,
		Username:     p.Username,
		PasswordHash: p.Password,
	})
	if err != nil {
		return View{}, apperr.NewDBError(err, User)
	}
	return NewView(row), nil
}

// ==========================================
// LIST
// ==========================================

func (s *Service) List(ctx context.Context, p ListParams) ([]View, error) {
	var pgCursor pgtype.UUID
	if p.Cursor != nil {
		pgCursor = pgtype.UUID{Bytes: *p.Cursor, Valid: true}
	}

	rows, err := s.store.UserList(ctx, repository.UserListParams{
		Column1: pgCursor,
		Limit:   p.Limit,
	})
	if err != nil {
		return nil, apperr.NewDBError(err, User)
	}

	views := make([]View, len(rows))
	for i, row := range rows {
		views[i] = NewView(row)
	}
	return views, nil
}

func (s *Service) ListUnverified(ctx context.Context, limit int32) ([]View, error) {
	rows, err := s.store.UserListUnverified(ctx, limit)
	if err != nil {
		return nil, apperr.NewDBError(err, User)
	}

	views := make([]View, len(rows))
	for i, row := range rows {
		views[i] = NewView(row)
	}
	return views, nil
}

// ==========================================
// GET
// ==========================================

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (View, error) {
	row, err := s.store.UserGetByID(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return View{}, apperr.NewDBError(err, User)
	}
	return NewView(row), nil
}

func (s *Service) GetByEmail(ctx context.Context, email string) (View, error) {
	row, err := s.store.UserGetByEmail(ctx, email)
	if err != nil {
		return View{}, apperr.NewDBError(err, User)
	}
	return NewView(row), nil
}

func (s *Service) GetByUsername(ctx context.Context, username string) (View, error) {
	row, err := s.store.UserGetByUsername(ctx, username)
	if err != nil {
		return View{}, apperr.NewDBError(err, User)
	}
	return NewView(row), nil
}

// ==========================================
// UPDATE
// ==========================================

func (s *Service) MarkVerified(ctx context.Context, id uuid.UUID) (View, error) {
	row, err := s.store.UserMarkVerified(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return View{}, apperr.NewDBError(err, User)
	}
	return NewView(row), nil
}

func (s *Service) UpdatePassword(ctx context.Context, p UpdatePasswordParams) (View, error) {
	row, err := s.store.UserUpdatePassword(ctx, repository.UserUpdatePasswordParams{
		ID:           pgtype.UUID{Bytes: p.ID, Valid: true},
		PasswordHash: p.Hash,
	})
	if err != nil {
		return View{}, apperr.NewDBError(err, User)
	}
	return NewView(row), nil
}

func (s *Service) UpdateLastVerificationSent(ctx context.Context, id uuid.UUID) (View, error) {
	row, err := s.store.UserUpdateLastVerificationSent(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return View{}, apperr.NewDBError(err, User)
	}
	return NewView(row), nil
}

func (s *Service) EnableTOTP(ctx context.Context, p EnableTOTPParams) (View, error) {
	row, err := s.store.UserEnableTOTP(ctx, repository.UserEnableTOTPParams{
		ID:         pgtype.UUID{Bytes: p.ID, Valid: true},
		TotpSecret: pgtype.Text{String: p.Secret, Valid: true},
	})
	if err != nil {
		return View{}, apperr.NewDBError(err, User)
	}
	return NewView(row), nil
}

func (s *Service) DisableTOTP(ctx context.Context, id uuid.UUID) (View, error) {
	row, err := s.store.UserDisableTOTP(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return View{}, apperr.NewDBError(err, User)
	}
	return NewView(row), nil
}

// ==========================================
// DELETE
// ==========================================

func (s *Service) DeleteByID(ctx context.Context, id uuid.UUID) error {
	err := s.store.UserDeleteByID(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return apperr.NewDBError(err, User)
	}
	return nil
}

func (s *Service) DeleteByEmail(ctx context.Context, email string) error {
	err := s.store.UserDeleteByEmail(ctx, email)
	if err != nil {
		return apperr.NewDBError(err, User)
	}
	return nil
}
