package presence

import (
	"bonfire-api/internal/cache"
	"context"
)

// --- presence service ---

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

// --- presence service Heartbeat ---

func (s *Service) Heartbeat(
	ctx context.Context,
	userID string,
) error {

	return s.cache.Heartbeat(ctx, userID)
}

// --- presence service GetActivity ---

func (s *Service) GetActivity(
	ctx context.Context,
	userID string,
) (Activity, error) {

	return s.cache.GetActivity(ctx, userID)
}

// --- presence service GetBulkActivity ---


func (s *Service) GetBulkActivity(
	ctx context.Context,
	userIDs []string,
) (map[string]Activity, error) {

	return s.cache.GetBulkActivity(ctx, userIDs)
}

// --- presence service UpdateStatus ---

type PresenceUpdatedEvent struct {
	UserID string   `json:"user_id"`
	Status Activity `json:"status"`
}

const PresenceUpdatedChannel = "presence.updated"

func (s *Service) UpdateStatus(
	ctx context.Context,
	userID string,
	status Activity,
) error {

	if err := s.cache.SetStatus(ctx, userID, status); err != nil {
		return err
	}

	event := PresenceUpdatedEvent{
		UserID: userID,
		Status: status,
	}

	return s.cache.Publish(
		ctx,
		PresenceUpdatedChannel,
		event,
	)
}
