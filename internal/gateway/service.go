package gateway

import (
	"context"
	"encoding/json"

	"bonfire-api/internal/errs"
	"bonfire-api/internal/fields"
	"bonfire-api/internal/presence"

	"github.com/google/uuid"
)

type Service struct {
	userCache     UserCache
	presenceCache PresenceCache
	pub           Publisher
}

func NewService(
	userCache UserCache,
	presenceCache PresenceCache,
	pub Publisher,
) *Service {
	return &Service{
		userCache:     userCache,
		presenceCache: presenceCache,
		pub:           pub,
	}
}

func (s *Service) RegisterNode(
	ctx context.Context,
	userID, nodeID, sessionID fields.ID,
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

func (s *Service) UnregisterNode(ctx context.Context, userID, nodeID, sessionID fields.ID) error {
	// wentOffline, err := s.presenceCache.UnregisterNode(ctx, userID, nodeID, sessionID)
	// if err != nil {
	// 	return err
	// }

	// if wentOffline {
	// 	payload := user.EventUpdatePresencePayload{
	// 		UserID:   userID.String(),
	// 		Presence: presence.NewOffline().String(),
	// 	}
	// 	if broadcastErr := s.BroadcastToPeers(ctx, userID, user.EventUpdatePresence, payload); broadcastErr != nil {
	// 		slog.ErrorContext(ctx, "failed to broadcast presence update on unregister", "user_id", userID, "error", broadcastErr)
	// 	}
	// }

	return nil
}

func (s *Service) HandleHeartbeat(
	ctx context.Context,
	userID, nodeID, sessionID fields.ID,
	newPresence presence.Presence,
) error {
	currentPresence, err := s.presenceCache.GetPresence(ctx, userID)
	if err != nil {
		return err
	}

	if currentPresence == presence.NewOffline() || (newPresence.IsValid() && newPresence != currentPresence) {
		return s.RegisterNode(ctx, userID, nodeID, sessionID, newPresence)
	}

	return s.presenceCache.Heartbeat(ctx, userID, nodeID, sessionID)
}

func (s *Service) RemoveBatchNodes(ctx context.Context, userIDs []fields.ID, nodeID fields.ID) error {
	if len(userIDs) == 0 {
		return nil
	}
	return s.presenceCache.RemoveBatchNodes(ctx, userIDs, nodeID)
}

func (s *Service) BroadcastToSession(
	ctx context.Context,
	actorID fields.ID,
	targetUserID fields.ID,
	targetSessionID fields.ID,
	eventType string,
	payload interface{},
) error {
	nodeID, found, err := s.presenceCache.GetSessionNode(ctx, targetSessionID)
	if err != nil || !found {
		return err
	}

	rawPayload, err := json.Marshal(payload)
	if err != nil {
		return errs.Internal("Failed to marshal event payload.").Wrap(err)
	}

	nodeEvents := map[fields.ID]Event{
		nodeID: {
			SessionIDs: []uuid.UUID{targetSessionID.UUID()},
			Type:       eventType,
			Data:       rawPayload,
		},
	}

	return s.pub.PublishEvents(ctx, nodeEvents)
}

func (s *Service) BroadcastUserEvent(
	ctx context.Context,
	recipientIDs []fields.ID,
	excludeSessionIDs []fields.ID,
	eventType string,
	payload interface{},
) error {
	if len(recipientIDs) == 0 {
		return nil
	}

	nodeToRecipients, err := s.presenceCache.GetBatchNodes(ctx, recipientIDs)
	if err != nil {
		return err
	}
	if len(nodeToRecipients) == 0 {
		return nil
	}

	rawPayload, err := json.Marshal(payload)
	if err != nil {
		return errs.Internal("Failed to marshal event payload.").Wrap(err)
	}

	nodeEvents := make(map[fields.ID]Event, len(nodeToRecipients))
	for nodeID, targetUserIDs := range nodeToRecipients {
		nodeEvents[nodeID] = Event{
			UserIDs:           fields.UUIDs(targetUserIDs),
			ExcludeSessionIDs: fields.UUIDs(excludeSessionIDs),
			Type:              eventType,
			Data:              rawPayload,
		}
	}

	return s.pub.PublishEvents(ctx, nodeEvents)
}

func (s *Service) BroadcastToUser(
	ctx context.Context,
	userID fields.ID,
	excludeSessionIDs []fields.ID,
	eventType string,
	payload interface{},
) error {
	return s.BroadcastUserEvent(ctx, []fields.ID{userID}, excludeSessionIDs, eventType, payload)
}

func (s *Service) BroadcastToUsers(
	ctx context.Context,
	recipientIDs []fields.ID,
	excludeSessionIDs []fields.ID,
	eventType string,
	payload interface{},
) error {
	return s.BroadcastUserEvent(ctx, recipientIDs, excludeSessionIDs, eventType, payload)
}

// func (s *Service) BroadcastToFriends(
// 	ctx context.Context,
// 	actorID fields.ID,
// 	eventType string,
// 	payload interface{},
// ) error {
// 	friendIDs, err := s.userCache.GetFriendIDs(ctx, actorID)
// 	if err != nil {
// 		return err
// 	}

// 	recipients := make([]fields.ID, 0, len(friendIDs)+1)
// 	recipients = append(recipients, friendIDs...)
// 	recipients = append(recipients, actorID)

// 	return s.BroadcastUserEvent(ctx, actorID, recipients, eventType, payload)
// }

// func (s *Service) BroadcastToPeers(
// 	ctx context.Context,
// 	actorID fields.ID,
// 	eventType string,
// 	payload interface{},
// ) error {
// 	peerIDs, err := s.userCache.GetPeerIDs(ctx, actorID)
// 	if err != nil {
// 		return err
// 	}

// 	recipients := make([]fields.ID, 0, len(peerIDs)+1)
// 	recipients = append(recipients, peerIDs...)
// 	recipients = append(recipients, actorID)

// 	return s.BroadcastUserEvent(ctx, actorID, recipients, eventType, payload)
// }

// func (s *Service) BroadcastToChannelMembers(
// 	ctx context.Context,
// 	actorID fields.ID,
// 	actorExcludeSessionIDs []fields.ID,
// 	eventType string,
// 	payloads map[fields.ID]interface{},
// ) error {

// 	// Get recipient ids (payloads keys)

// 	// get batch nodes for the users

// 	// create node events
// }

/*
BroadcastToChannelMembers

(
	ctx context.Context,
	actorID fields.ID,
	actorSessionID fields.ID,
	eventType string,
	payloads interface{},
)

// get member ids
// make recipients
// broadcast
*/

// type Event struct {
//     UserIDs    []uuid.UUID     `json:"user_ids,omitempty"`
//     SessionIDs []uuid.UUID     `json:"session_ids,omitempty"`
//     Type       string          `json:"type"`
//     Data       json.RawMessage `json:"data"`
// }
