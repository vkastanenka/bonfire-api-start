package gateway

import (
	"context"

	"bonfire-api/internal/presence"

	"github.com/google/uuid"
)

type SessionManager struct {
	userCache     UserCache
	presenceCache PresenceCache
}

func NewSessionManager(
	userCache UserCache,
	presenceCache PresenceCache,
) *SessionManager {
	return &SessionManager{
		userCache:     userCache,
		presenceCache: presenceCache,
	}
}

func (s *SessionManager) RegisterNode(
	ctx context.Context,
	userID, nodeID, sessionID uuid.UUID,
	presenceStatus presence.Presence,
) error {
	// wasOffline, effPresence, err := s.presenceCache.RegisterNode(ctx, userID, nodeID, sessionID, presenceStatus)
	// if err != nil {
	// 	return err
	// }

	// if wasOffline {
	// 	payload := user.EventUpdatePresencePayload{
	// 		UserID:   userID.String(),
	// 		Presence: effPresence.String(),
	// 	}
	// 	if broadcastErr := s.BroadcastToPeers(ctx, userID, user.EventUpdatePresence, payload); broadcastErr != nil {
	// 		slog.ErrorContext(ctx, "failed to broadcast presence update on register", "user_id", userID, "error", broadcastErr)
	// 	}
	// }

	return nil
}

func (s *SessionManager) UnregisterNode(ctx context.Context, userID, nodeID, sessionID uuid.UUID) error {
	// wentOffline, err := s.presenceCache.UnregisterNode(ctx, userID, nodeID, sessionID)
	// if err != nil {
	// 	return err
	// }

	// if wentOffline {
	// 	payload := user.EventUpdatePresencePayload{
	// 		UserID:   userID.String(),
	// 		Presence: presence.PresenceOffline.String(),
	// 	}
	// 	if broadcastErr := s.BroadcastToPeers(ctx, userID, user.EventUpdatePresence, payload); broadcastErr != nil {
	// 		slog.ErrorContext(ctx, "failed to broadcast presence update on unregister", "user_id", userID, "error", broadcastErr)
	// 	}
	// }

	return nil
}

func (s *SessionManager) HandleHeartbeat(
	ctx context.Context,
	userID, nodeID, sessionID uuid.UUID,
	newPresence presence.Presence,
) error {
	currentPresence, err := s.presenceCache.GetPresence(ctx, userID)
	if err != nil {
		return err
	}

	if currentPresence == presence.PresenceOffline || (newPresence.IsValid() && newPresence != currentPresence) {
		return s.RegisterNode(ctx, userID, nodeID, sessionID, newPresence)
	}

	return s.presenceCache.Heartbeat(ctx, userID, nodeID, sessionID)
}

func (s *SessionManager) RemoveBatchNodes(ctx context.Context, userIDs []uuid.UUID, nodeID uuid.UUID) error {
	if len(userIDs) == 0 {
		return nil
	}
	return s.presenceCache.RemoveBatchNodeUsers(ctx, nodeID, userIDs)
}
