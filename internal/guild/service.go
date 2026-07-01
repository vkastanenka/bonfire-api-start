package guild

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/repository"
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type Service struct {
	store repository.Store
}

func NewService(store repository.Store) *Service {
	return &Service{store: store}
}

// GetFull fetches the joined guild and profile data
func (s *Service) GetFull(ctx context.Context, id uuid.UUID) (View, error) {
	row, err := s.store.GuildGetFull(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return View{}, apperr.NewDBError(err)
	}
	return NewView(row), nil
}

// GetEffectivePermissions aggregates bitwise permissions
func (s *Service) GetEffectivePermissions(ctx context.Context, guildID, userID uuid.UUID) (int64, error) {
	perm, err := s.store.GetEffectivePermissions(ctx, repository.GetEffectivePermissionsParams{
		GuildID: pgtype.UUID{Bytes: guildID, Valid: true},
		UserID:  pgtype.UUID{Bytes: userID, Valid: true},
	})
	if err != nil {
		return 0, apperr.NewDBError(err)
	}
	return perm, nil
}

// ListMembers gets all users in a guild
func (s *Service) ListMembers(ctx context.Context, guildID uuid.UUID) ([]MemberView, error) {
	rows, err := s.store.GuildListMembers(ctx, pgtype.UUID{Bytes: guildID, Valid: true})
	if err != nil {
		return nil, apperr.NewDBError(err)
	}

	members := make([]MemberView, len(rows))
	for i, r := range rows {
		members[i] = MemberView{ID: r.ID.Bytes, Username: r.Username}
	}
	return members, nil
}
