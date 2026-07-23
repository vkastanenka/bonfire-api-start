package presence

import (
	"context"

	"bonfire-api/internal/errs"

	"github.com/google/uuid"
)

type Store interface {
	SetPresence(ctx context.Context, userID uuid.UUID, p Presence) error
	GetPresence(ctx context.Context, userID uuid.UUID) (Presence, error)
	GetPresenceBulk(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]Presence, error)
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{
		store: store,
	}
}

func (s *Service) Set(ctx context.Context, userID uuid.UUID, rawPresence string) error {
	if userID == uuid.Nil {
		return errs.InvalidArgument("User ID cannot be empty.").
			FieldViolation("user_id", "User ID is required.", "INVALID_USER_ID")
	}

	p, err := New(rawPresence)
	if err != nil {
		return errs.InvalidArgument("Invalid presence status provided.").
			FieldViolation("presence", "Value must be one of: online, offline, idle, away, busy, or invisible.", "INVALID_PRESENCE_STATUS").Wrap(err)
	}

	return s.store.SetPresence(ctx, userID, p)
}

func (s *Service) Get(ctx context.Context, userID uuid.UUID) (Presence, error) {
	if userID == uuid.Nil {
		return PresenceOffline, errs.InvalidArgument("User ID cannot be empty.").
			FieldViolation("user_id", "User ID is required.", "INVALID_USER_ID")
	}

	return s.store.GetPresence(ctx, userID)
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

	return s.store.GetPresenceBulk(ctx, userIDs)
}
