package gateway

import (
	"bonfire-api/internal/presence"
	"context"
	"log/slog"

	"github.com/google/uuid"
)

const (
	EventUpdatePresence = "user.update_presence"
)

type EventUpdatePresencePayload struct {
	UserID   uuid.UUID         `json:"userId"`
	Presence presence.Presence `json:"presence"`
}

type Service struct {
	broadcaster   *Broadcaster
	relationCache RelationCache
	gatewayCache  GatewayCache
	presenceCache PresenceCache
	userCache     UserCache
}

func NewService(
	broadcaster *Broadcaster,
	relationCache RelationCache,
	gatewayCache GatewayCache,
	presenceCache PresenceCache,
	userCache UserCache,
) *Service {
	return &Service{
		broadcaster:   broadcaster,
		relationCache: relationCache,
		gatewayCache:  gatewayCache,
		presenceCache: presenceCache,
		userCache:     userCache,
	}
}

func (s *Service) RegisterSession(
	ctx context.Context,
	nodeID, userID, sessionID uuid.UUID,
	p presence.Presence,
) error {
	wasOffline, err := s.gatewayCache.RegisterSession(ctx, nodeID, userID, sessionID, p)
	if err != nil {
		return err
	}

	if wasOffline {
		friendIDs, err := s.relationCache.GetFriendIDs(ctx, userID)
		if err != nil {
			return err
		}

		if len(friendIDs) == 0 {
			return nil
		}

		payload := EventUpdatePresencePayload{
			UserID:   userID,
			Presence: p,
		}

		broadcastErr := s.broadcaster.BroadcastToUsers(ctx, friendIDs, EventUpdatePresence, payload)
		if broadcastErr != nil {
			slog.ErrorContext(ctx, "failed to broadcast presence update on register", "user_id", userID, "error", broadcastErr)
		}
	}

	return nil
}

func (s *Service) UnregisterSession(ctx context.Context, nodeID, userID, sessionID uuid.UUID) error {
	wentOffline, err := s.gatewayCache.UnregisterSession(ctx, nodeID, userID, sessionID)
	if err != nil {
		return err
	}

	if wentOffline {
		friendIDs, err := s.relationCache.GetFriendIDs(ctx, userID)
		if err != nil {
			return err
		}

		if len(friendIDs) == 0 {
			return nil
		}

		payload := EventUpdatePresencePayload{
			UserID:   userID,
			Presence: presence.PresenceOffline,
		}

		broadcastErr := s.broadcaster.BroadcastToUsers(ctx, friendIDs, EventUpdatePresence, payload)
		if broadcastErr != nil {
			slog.ErrorContext(ctx, "failed to broadcast presence update on unregister", "user_id", userID, "error", broadcastErr)
		}
	}

	return nil
}

func (s *Service) HandleHeartbeat(
	ctx context.Context,
	userID, nodeID, sessionID uuid.UUID,
	newPresence presence.Presence,
) error {
	currentPresence, err := s.presenceCache.Get(ctx, userID)
	if err != nil {
		return err
	}

	if currentPresence == presence.PresenceOffline || (newPresence.IsValid() && newPresence != currentPresence) {
		if !newPresence.IsValid() {
			newPresence = presence.PresenceOnline
		}
		return s.RegisterSession(ctx, userID, nodeID, sessionID, newPresence)
	}

	return s.gatewayCache.Heartbeat(ctx, userID, nodeID, sessionID)
}

func (s *Service) RemoveBatchUsers(ctx context.Context, nodeID uuid.UUID, userIDs []uuid.UUID) error {
	return s.gatewayCache.RemoveBatchUsers(ctx, nodeID, userIDs)
}
