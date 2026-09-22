package presence

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
)

type Broadcaster interface {
	BroadcastToUser(ctx context.Context, recipientID uuid.UUID, eventType string, payload any, excludeSessionIDs ...uuid.UUID) error
	BroadcastToUsers(ctx context.Context, recipientIDs []uuid.UUID, eventType string, payload any, excludeSessionIDs ...uuid.UUID) error
}

const (
	EventUpdatePresence = "user.update_presence"
)

type EventUpdatePresencePayload struct {
	UserID   uuid.UUID `json:"user_id"`
	Presence Presence  `json:"presence"`
}

type Service struct {
	broadcaster   Broadcaster
	userCache     UserCache
	presenceCache PresenceCache
}

func NewService(
	broadcaster Broadcaster,
	userCache UserCache,
	presenceCache PresenceCache,
) *Service {
	return &Service{
		broadcaster:   broadcaster,
		userCache:     userCache,
		presenceCache: presenceCache,
	}
}

func (s *Service) RegisterNodeSession(
	ctx context.Context,
	userID, nodeID, sessionID uuid.UUID,
	presenceStatus Presence,
) error {
	wasOffline, effPresence, err := s.presenceCache.RegisterNodeSession(ctx, userID, nodeID, sessionID, presenceStatus)
	if err != nil {
		return err
	}

	if wasOffline {
		peerIDs, err := s.userCache.GetPeerIDs(ctx, userID)
		if err != nil {
			return err
		}

		if len(peerIDs) == 0 {
			return nil
		}

		payload := EventUpdatePresencePayload{
			UserID:   userID,
			Presence: effPresence,
		}
		if broadcastErr := s.broadcaster.BroadcastToUsers(ctx, peerIDs, EventUpdatePresence, payload); broadcastErr != nil {
			slog.ErrorContext(ctx, "failed to broadcast presence update on register", "user_id", userID, "error", broadcastErr)
		}
	}

	return nil
}

func (s *Service) UnregisterNodeSession(ctx context.Context, userID, nodeID, sessionID uuid.UUID) error {
	wentOffline, err := s.presenceCache.UnregisterNodeSession(ctx, userID, nodeID, sessionID)
	if err != nil {
		return err
	}

	if wentOffline {
		peerIDs, err := s.userCache.GetPeerIDs(ctx, userID)
		if err != nil {
			return err
		}

		if len(peerIDs) == 0 {
			return nil
		}

		payload := EventUpdatePresencePayload{
			UserID:   userID,
			Presence: PresenceOffline,
		}
		if broadcastErr := s.broadcaster.BroadcastToUsers(ctx, peerIDs, EventUpdatePresence, payload); broadcastErr != nil {
			slog.ErrorContext(ctx, "failed to broadcast presence update on unregister", "user_id", userID, "error", broadcastErr)
		}
	}

	return nil
}

func (s *Service) RemoveBatchNodeUsers(ctx context.Context, nodeID uuid.UUID, userIDs []uuid.UUID) error {
	if len(userIDs) == 0 {
		return nil
	}
	return s.presenceCache.RemoveBatchNodeUsers(ctx, nodeID, userIDs)
}

func (s *Service) HandleHeartbeat(
	ctx context.Context,
	userID, nodeID, sessionID uuid.UUID,
	newPresence Presence,
) error {
	currentPresence, err := s.presenceCache.GetPresence(ctx, userID)
	if err != nil {
		return err
	}

	if currentPresence == PresenceOffline || (newPresence.IsValid() && newPresence != currentPresence) {
		if !newPresence.IsValid() {
			newPresence = PresenceOnline
		}
		return s.RegisterNodeSession(ctx, userID, nodeID, sessionID, newPresence)
	}

	return s.presenceCache.Heartbeat(ctx, userID, nodeID, sessionID)
}
