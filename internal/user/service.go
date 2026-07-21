package user

import (
	"context"
	"fmt"

	"bonfire-api/internal/apperr"
	"bonfire-api/internal/cache"
	"bonfire-api/internal/db"

	"github.com/google/uuid"
)

func userPresenceKey(userID uuid.UUID) string {
	return fmt.Sprintf("user:{%s}:presence", userID.String())
}

type PresenceService interface {
	Heartbeat(ctx context.Context, userID uuid.UUID, p Presence) error
	GetPresenceByUserID(ctx context.Context, userID uuid.UUID) (Presence, error)
	GetPresenceBulkByUserIDs(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]Presence, error)
}

type Service struct {
	cache cache.Manager
	db    db.Store
}

func NewService(cache cache.Manager) *Service {
	return &Service{
		cache: cache,
	}
}

var _ PresenceService = (*Service)(nil)

func (s *Service) Heartbeat(ctx context.Context, userID uuid.UUID, p Presence) error {
	if !p.IsValid() {
		return apperr.NewInvalidArgument(
			nil,
			apperr.WithMsg("Invalid presence status provided"),
			apperr.WithFieldViolation("presence", fmt.Sprintf("presence value %d is invalid", p), "INVALID_ENUM_VALUE"),
		)
	}

	key := userPresenceKey(userID)
	if err := s.cache.Set(ctx, key, p.String(), presenceTTL); err != nil {
		return apperr.NewInternal(
			err,
			apperr.WithMsg("Failed to record user presence heartbeat"),
			apperr.WithMeta("user_id", userID.String()),
		)
	}

	return nil
}

func (s *Service) GetPresenceByUserID(ctx context.Context, userID uuid.UUID) (Presence, error) {
	var val string
	err := s.cache.Get(ctx, userPresenceKey(userID), &val)

	if cache.IsNotFoundError(err) {
		return PresenceOffline, nil
	}

	if err != nil {
		return PresenceUnknown, apperr.NewInternal(
			err,
			apperr.WithMsg("Failed to retrieve user presence"),
			apperr.WithMeta("user_id", userID.String()),
		)
	}

	return NewPresence(val)
}

func (s *Service) GetPresenceBulkByUserIDs(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]Presence, error) {
	activities := make(map[uuid.UUID]Presence, len(userIDs))
	if len(userIDs) == 0 {
		return activities, nil
	}

	// presenceKeys := make([]string, len(userIDs))
	// for i, id := range userIDs {
	// 	presenceKeys[i] = userPresenceKey(id)
	// }

	// values, err := s.cache.MGet(ctx, presenceKeys...)
	// if err != nil {
	// 	return nil, apperr.NewInternal(
	// 		err,
	// 		apperr.WithMsg("Failed to bulk retrieve user presences"),
	// 		apperr.WithMeta("requested_count", fmt.Sprintf("%d", len(userIDs))),
	// 	)
	// }

	// for i, id := range userIDs {
	// 	if values[i] == nil {
	// 		activities[id] = PresenceOffline
	// 		continue
	// 	}

	// 	if valStr, ok := values[i].(string); ok {
	// 		activities[id] = NewPresence(valStr)
	// 	} else {
	// 		activities[id] = PresenceOffline
	// 	}
	// }

	return activities, nil
}

// func (s *Service) GetMe(ctx context.Context, userID uuid.UUID) (View, error) {
// 	var (
// 		u  User
// 		up UserProfile
// 	)

// 	g, gCtx := errgroup.WithContext(ctx)

// 	g.Go(func() error {
// 		var err error
// 		u, err = s.db.UserGetByID(gCtx, userID)
// 		return err
// 	})

// 	g.Go(func() error {
// 		var err error
// 		up, err = s.GetProfileByUserID(gCtx, userID)
// 		return err
// 	})

// 	if err := g.Wait(); err != nil {
// 		return View{}, err
// 	}

// 	return ToView(u, up), nil
// }
