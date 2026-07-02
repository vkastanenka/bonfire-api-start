package presence

import (
	"bonfire-api/internal/redis"
	"context"
)

// --- presence service ---

type Service struct {
	redis redis.Manager
}

func NewService(
	redis redis.Manager,
) *Service {
	return &Service{
		redis: redis,
	}
}

// --- presence service Heartbeat ---

func (s *Service) Heartbeat(
	ctx context.Context,
	userID string,
) error {

	return s.redis.Heartbeat(ctx, userID)
}

// --- presence service GetActivity ---

func (s *Service) GetActivity(
	ctx context.Context,
	userID string,
) (Activity, error) {

	return s.redis.GetActivity(ctx, userID)
}

// --- presence service GetBulkActivity ---

func (s *Service) GetBulkActivity(
	ctx context.Context,
	userIDs []string,
) (map[string]Activity, error) {

	return s.redis.GetBulkActivity(ctx, userIDs)
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

	if err := s.redis.SetStatus(ctx, userID, status); err != nil {
		return err
	}

	event := PresenceUpdatedEvent{
		UserID: userID,
		Status: status,
	}

	return s.redis.Publish(
		ctx,
		PresenceUpdatedChannel,
		event,
	)
}
