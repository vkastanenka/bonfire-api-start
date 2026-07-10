package presence

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/cache"
	"context"
	"fmt"

	"github.com/google/uuid"
)

func userPresenceKey(userID uuid.UUID) string {
	return fmt.Sprintf("user:{%s}:presence", userID.String())
}

type Service struct {
	cache cache.Manager
}

func NewService(
	cache cache.Manager,
) *Service {
	return &Service{
		cache: cache,
	}
}

func (s *Service) Heartbeat(ctx context.Context, userID uuid.UUID, p Presence) error {
	if !p.Valid() {
		return apperr.NewInvalidInput(nil, fmt.Sprintf("invalid presence: %d", p))
	}
	key := userPresenceKey(userID)

	return s.cache.Set(ctx, key, p.String(), presenceTTL)
}

func (s *Service) GetByUserID(ctx context.Context, userID uuid.UUID) (Presence, error) {
	var val string
	err := s.cache.Get(ctx, userPresenceKey(userID), &val)

	if cache.IsNotFoundError(err) {
		return PresenceOffline, nil
	}

	if err != nil {
		return PresenceUnknown, err
	}

	return ParsePresence(val), nil
}

func (s *Service) GetBulkByUserIDs(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]Presence, error) {
	activities := make(map[uuid.UUID]Presence, len(userIDs))
	if len(userIDs) == 0 {
		return activities, nil
	}

	presenceKeys := make([]string, len(userIDs))
	for i, id := range userIDs {
		presenceKeys[i] = userPresenceKey(id)
	}

	values, err := s.cache.MGet(ctx, presenceKeys...)
	if err != nil {
		return nil, err
	}

	for i, id := range userIDs {
		if values[i] == nil {
			activities[id] = PresenceOffline
			continue
		}

		if valStr, ok := values[i].(string); ok {
			activities[id] = ParsePresence(valStr)
		} else {
			activities[id] = PresenceOffline
		}
	}

	return activities, nil
}
