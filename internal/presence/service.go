package presence

import (
	"context"

	"bonfire-api/internal/errs"

	"github.com/google/uuid"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
	}
}

func (s *Service) Set(ctx context.Context, userID uuid.UUID, p Presence) error {
	// Guard: Nil/Zero UUID check
	if userID == uuid.Nil {
		return errs.InvalidArgument("User ID cannot be empty.").
			FieldViolation("user_id", "User ID is required.", "INVALID_USER_ID")
	}

	// Guard: Domain invariant validation
	if !p.IsValid() {
		return errs.InvalidArgument("Invalid presence status provided.").
			FieldViolation("presence", "Value must be one of: online, offline, away, or busy.", "INVALID_PRESENCE_STATUS")
	}

	return s.repo.SetPresence(ctx, userID, p)
}

func (s *Service) Get(ctx context.Context, userID uuid.UUID) (Presence, error) {
	if userID == uuid.Nil {
		return PresenceOffline, errs.InvalidArgument("User ID cannot be empty.").
			FieldViolation("user_id", "User ID is required.", "INVALID_USER_ID")
	}

	return s.repo.GetPresence(ctx, userID)
}

func (s *Service) GetBulk(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]Presence, error) {
	if len(userIDs) == 0 {
		return make(map[uuid.UUID]Presence), nil
	}

	for _, id := range userIDs {
		if id == uuid.Nil {
			return nil, errs.InvalidArgument("User IDs slice contains an empty UUID.").
				FieldViolation("user_ids", "User IDs cannot contain empty values.", "INVALID_USER_ID")
		}
	}

	return s.repo.GetPresenceBulk(ctx, userIDs)
}
