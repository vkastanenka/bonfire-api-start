package gateway

import (
	"context"

	"github.com/google/uuid"
)

const (
	EventUpdatePresence = "user.update_presence"
)

type EventUpdatePresencePayload struct {
	UserID   uuid.UUID `json:"userId"`
	Presence Presence  `json:"presence"`
}

type Service struct {
	cache       *Cache
	broadcaster *Broadcaster
	// presenceCache PresenceCache
	// relationCache RelationCache
	// userCache     UserCache
}

func NewService(
	cache *Cache,
	broadcaster *Broadcaster,
	// presenceCache PresenceCache,
	// relationCache RelationCache,
	// userCache UserCache,
) *Service {
	return &Service{
		cache:       cache,
		broadcaster: broadcaster,
		// presenceCache: presenceCache,
		// relationCache: relationCache,
		// userCache:     userCache,
	}
}

type PrintTicketParams struct {
	UserID    uuid.UUID
	SessionID uuid.UUID
}

func (s *Service) PrintTicket(ctx context.Context, p PrintTicketParams) (uuid.UUID, error) {
	ticket := uuid.New()
	clientID := uuid.New()

	if err := s.cache.PrintTicket(ctx, ticket, p.UserID, p.SessionID, clientID); err != nil {
		return uuid.UUID{}, err
	}

	return ticket, nil
}

func (s *Service) PunchTicket(ctx context.Context, id uuid.UUID) (uuid.UUID, uuid.UUID, uuid.UUID, error) {
	return s.cache.PunchTicket(ctx, id)
}

func (s *Service) RegisterClient(
	ctx context.Context,
	hubID, userID, sessionID, clientID uuid.UUID,
	p Presence,
) error {
	_, err := s.cache.RegisterClient(ctx, hubID, userID, sessionID, clientID, p)
	if err != nil {
		return err
	}

	// if wasOffline {
	// 	friendIDs, err := s.relationCache.GetFriendIDs(ctx, userID)
	// 	if err != nil {
	// 		return err
	// 	}

	// 	if len(friendIDs) == 0 {
	// 		return nil
	// 	}

	// 	payload := EventUpdatePresencePayload{
	// 		UserID:   userID,
	// 		Presence: p,
	// 	}

	// 	broadcastErr := s.broadcaster.BroadcastToUsers(ctx, friendIDs, EventUpdatePresence, payload)
	// 	if broadcastErr != nil {
	// 		slog.ErrorContext(ctx, "failed to broadcast presence update on register", "user_id", userID, "error", broadcastErr)
	// 	}
	// }

	return nil
}

func (s *Service) UnregisterClient(ctx context.Context, hubID, userID, sessionID, clientID uuid.UUID) error {
	_, err := s.cache.UnregisterClient(ctx, hubID, userID, sessionID, clientID)
	if err != nil {
		return err
	}

	// if wentOffline {
	// 	friendIDs, err := s.relationCache.GetFriendIDs(ctx, userID)
	// 	if err != nil {
	// 		return err
	// 	}

	// 	if len(friendIDs) == 0 {
	// 		return nil
	// 	}

	// 	payload := EventUpdatePresencePayload{
	// 		UserID:   userID,
	// 		Presence: PresenceOffline,
	// 	}

	// 	broadcastErr := s.broadcaster.BroadcastToUsers(ctx, friendIDs, EventUpdatePresence, payload)
	// 	if broadcastErr != nil {
	// 		slog.ErrorContext(ctx, "failed to broadcast presence update on unregister", "user_id", userID, "error", broadcastErr)
	// 	}
	// }

	return nil
}

// func (s *Service) HandleHeartbeat(
// 	ctx context.Context,
// 	userID, nodeID, sessionID uuid.UUID,
// 	newPresence Presence,
// ) error {
// 	currentPresence, err := s.presenceCache.Get(ctx, userID)
// 	if err != nil {
// 		return err
// 	}

// 	if currentPresence == PresenceOffline || (newPresence.IsValid() && newPresence != currentPresence) {
// 		if !newPresence.IsValid() {
// 			newPresence = PresenceOnline
// 		}
// 		return s.RegisterSession(ctx, userID, nodeID, sessionID, newPresence)
// 	}

// 	return s.gatewayCache.Heartbeat(ctx, userID, nodeID, sessionID)
// }

// func (s *Service) RemoveBatchUsers(ctx context.Context, nodeID uuid.UUID, userIDs []uuid.UUID) error {
// 	return s.gatewayCache.RemoveBatchUsers(ctx, nodeID, userIDs)
// }
