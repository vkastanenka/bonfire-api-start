package user

import (
	"context"
	"fmt"

	"bonfire-api/internal/apperr"

	"github.com/google/uuid"
)

type Service struct {
	userRepo     Repository
	presenceRepo PresenceRepository
}

func NewService(userRepo Repository, presenceRepo PresenceRepository) *Service {
	return &Service{
		userRepo:     userRepo,
		presenceRepo: presenceRepo,
	}
}

func (s *Service) Heartbeat(ctx context.Context, userID uuid.UUID, p Presence) error {
	if !p.IsValid() {
		return apperr.NewInvalidArgument(
			nil,
			apperr.WithMsg("Invalid presence status provided"),
			apperr.WithFieldViolation("presence", fmt.Sprintf("presence value %d is invalid", p), "INVALID_ENUM_VALUE"),
		)
	}

	return s.presenceRepo.SetPresence(ctx, userID, p)
}

func (s *Service) GetPresenceByUserID(ctx context.Context, userID uuid.UUID) (Presence, error) {
	return s.presenceRepo.GetPresence(ctx, userID)
}

func (s *Service) GetPresenceBulkByUserIDs(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]Presence, error) {
	return s.presenceRepo.GetPresenceBulk(ctx, userIDs)
}
