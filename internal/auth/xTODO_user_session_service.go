package auth

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

func (s *Service) Count(ctx context.Context) (int64, error) {
	count, err := s.store.UserSessionCount(ctx)
	if err != nil {
		return 0, apperr.NewDBError(err)
	}
	return count, nil
}

// ==========================================
// CREATE
// ==========================================

func (s *Service) CreateUserSession(ctx context.Context, p CreateUserSessionParams) (UserSessionView, error) {
	row, err := s.store.UserSessionCreate(ctx, repository.UserSessionCreateParams{
		ID:           pgtype.UUID{Bytes: p.ID, Valid: true},
		UserID:       pgtype.UUID{Bytes: p.UserID, Valid: true},
		RefreshToken: p.RefreshToken,
		UserAgent:    p.UserAgent,
		ClientIP:     p.ClientIP,
		IsBlocked:    p.IsBlocked,
		ExpiresAt:    pgtype.Timestamptz{Time: p.ExpiresAt, Valid: true},
	})
	if err != nil {
		return UserSessionView{}, apperr.NewDBError(err)
	}
	return NewUserSessionView(row), nil
}

// ==========================================
// LIST
// ==========================================

func (s *Service) ListActiveUserSessionByUserID(ctx context.Context, userID uuid.UUID) ([]UserSessionView, error) {
	rows, err := s.store.UserSessionListActiveByUserID(ctx, pgtype.UUID{Bytes: userID, Valid: true})
	if err != nil {
		return nil, apperr.NewDBError(err)
	}

	views := make([]UserSessionView, len(rows))
	for i, row := range rows {
		views[i] = NewUserSessionView(row)
	}
	return views, nil
}

// ==========================================
// GET
// ==========================================

func (s *Service) GetUserSessionByID(ctx context.Context, id uuid.UUID) (UserSessionView, error) {
	row, err := s.store.UserSessionGetByID(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return UserSessionView{}, apperr.NewDBError(err)
	}
	return NewUserSessionView(row), nil
}

func (s *Service) GetUserSessionByRefreshToken(ctx context.Context, refreshToken string) (UserSessionView, error) {
	row, err := s.store.UserSessionGetByRefreshToken(ctx, refreshToken)
	if err != nil {
		return UserSessionView{}, apperr.NewDBError(err)
	}
	return NewUserSessionView(row), nil
}

// ==========================================
// UPDATE
// ==========================================

func (s *Service) UpdateUserSessionRefreshToken(ctx context.Context, p UpdateRefreshTokenParams) (UserSessionView, error) {
	row, err := s.store.UserSessionUpdateRefreshToken(ctx, repository.UserSessionUpdateRefreshTokenParams{
		ID:           pgtype.UUID{Bytes: p.ID, Valid: true},
		RefreshToken: p.RefreshToken,
		ExpiresAt:    pgtype.Timestamptz{Time: p.ExpiresAt, Valid: true},
	})
	if err != nil {
		return UserSessionView{}, apperr.NewDBError(err)
	}
	return NewUserSessionView(row), nil
}

func (s *Service) UpdateUserSessionLastSeen(ctx context.Context, id uuid.UUID) (UserSessionView, error) {
	row, err := s.store.UserSessionUpdateLastSeen(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return UserSessionView{}, apperr.NewDBError(err)
	}
	return NewUserSessionView(row), nil
}

func (s *Service) MarkUserSessionBlocked(ctx context.Context, id uuid.UUID) (UserSessionView, error) {
	row, err := s.store.UserSessionMarkBlocked(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return UserSessionView{}, apperr.NewDBError(err)
	}
	return NewUserSessionView(row), nil
}

// ==========================================
// DELETE
// ==========================================

func (s *Service) DeleteUserSession(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	err := s.store.UserSessionDelete(ctx, repository.UserSessionDeleteParams{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		UserID: pgtype.UUID{Bytes: userID, Valid: true},
	})
	if err != nil {
		return apperr.NewDBError(err)
	}
	return nil
}

func (s *Service) DeleteAllUserSessionExcept(ctx context.Context, userID uuid.UUID, exceptID uuid.UUID) error {
	err := s.store.UserSessionDeleteAllExcept(ctx, repository.UserSessionDeleteAllExceptParams{
		UserID: pgtype.UUID{Bytes: userID, Valid: true},
		ID:     pgtype.UUID{Bytes: exceptID, Valid: true},
	})
	if err != nil {
		return apperr.NewDBError(err)
	}
	return nil
}

func (s *Service) PurgeExpiredUserSession(ctx context.Context) error {
	err := s.store.UserSessionPurgeExpired(ctx)
	if err != nil {
		return apperr.NewDBError(err)
	}
	return nil
}
