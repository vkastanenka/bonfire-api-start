package gateway

import (
	"context"
	"log/slog"

	"bonfire-api/internal/presence"
	"bonfire-api/internal/user"

	"github.com/google/uuid"
)

type SessionManager struct {
	broadcaster   Broadcaster
	userCache     UserCache
	presenceCache PresenceCache
}

func NewSessionManager(
	broadcaster Broadcaster,
	userCache UserCache,
	presenceCache PresenceCache,
) *SessionManager {
	return &SessionManager{
		broadcaster:   broadcaster,
		userCache:     userCache,
		presenceCache: presenceCache,
	}
}

func (s *SessionManager) RegisterNodeSession(
	ctx context.Context,
	userID, nodeID, sessionID uuid.UUID,
	presenceStatus presence.Presence,
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

		payload := user.EventUpdatePresencePayload{
			UserID:   userID,
			Presence: effPresence,
		}
		if broadcastErr := s.broadcaster.BroadcastToUsers(ctx, peerIDs, user.EventUpdatePresence, payload); broadcastErr != nil {
			slog.ErrorContext(ctx, "failed to broadcast presence update on register", "user_id", userID, "error", broadcastErr)
		}
	}

	return nil
}

func (s *SessionManager) UnregisterNodeSession(ctx context.Context, userID, nodeID, sessionID uuid.UUID) error {
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

		payload := user.EventUpdatePresencePayload{
			UserID:   userID,
			Presence: presence.PresenceOffline,
		}
		if broadcastErr := s.broadcaster.BroadcastToUsers(ctx, peerIDs, user.EventUpdatePresence, payload); broadcastErr != nil {
			slog.ErrorContext(ctx, "failed to broadcast presence update on unregister", "user_id", userID, "error", broadcastErr)
		}
	}

	return nil
}

func (s *SessionManager) RemoveBatchNodeUsers(ctx context.Context, nodeID uuid.UUID, userIDs []uuid.UUID) error {
	if len(userIDs) == 0 {
		return nil
	}
	return s.presenceCache.RemoveBatchNodeUsers(ctx, nodeID, userIDs)
}

// func (s *SessionManager) HandleHeartbeat(
// 	ctx context.Context,
// 	userID, nodeID, sessionID uuid.UUID,
// 	newPresence presence.Presence,
// ) error {
// 	currentPresence, err := s.presenceCache.GetPresence(ctx, userID)
// 	if err != nil {
// 		return err
// 	}

// 	if currentPresence == presence.PresenceOffline || (newPresence.IsValid() && newPresence != currentPresence) {
// 		return s.RegisterNode(ctx, userID, nodeID, sessionID, newPresence)
// 	}

// 	return s.presenceCache.Heartbeat(ctx, userID, nodeID, sessionID)
// }
