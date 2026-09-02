package gateway

import (
	"bonfire-api/internal/errs"
	"bonfire-api/internal/fields"
	"bonfire-api/internal/user"
	"context"
	"encoding/json"
	"log/slog"
)

type Service struct {
	repo  UserRepository
	cache UserCache
	tx    TX
	pub   Publisher
}

func NewService(
	repo UserRepository,
	cache UserCache,
	tx TX,
	pub Publisher,
) *Service {
	return &Service{
		repo:  repo,
		cache: cache,
		tx:    tx,
		pub:   pub,
	}
}

func (s *Service) RegisterWSConnection(ctx context.Context, userID, connID fields.ID, initialPresence user.Presence) error {
	res, err := s.cache.RegisterWSConnection(ctx, userID, connID, initialPresence)
	if err != nil {
		return err
	}

	if res.WasOffline {
		s.pubUpdatePresence(ctx, userID, res.Presence)
	}

	return nil
}

func (s *Service) UnregisterWSConnection(ctx context.Context, userID, connID fields.ID) error {
	wentOffline, err := s.cache.UnregisterWSConnection(ctx, userID, connID)
	if err != nil {
		return err
	}

	if wentOffline {
		offlinePresence := user.NewPresenceOffline()
		s.pubUpdatePresence(ctx, userID, offlinePresence)
	}
	
	return nil
}

func (s *Service) RemoveBatchNode(ctx context.Context, userIDs []fields.ID, nodeID fields.ID) error {
	if len(userIDs) == 0 {
		return nil
	}
	return s.cache.RemoveBatchNode(ctx, userIDs, nodeID)
}

func (s *Service) HandleHeartbeat(ctx context.Context, userID, connID fields.ID, newPresence user.Presence) error {
	currentPresence, err := s.cache.GetPresence(ctx, userID)
	if err != nil {
		return err
	}

	if currentPresence == user.NewPresenceOffline() {
		err := s.RegisterWSConnection(ctx, userID, connID, newPresence)
		return err
	}

	if newPresence.IsValid() && newPresence != currentPresence {
		err := s.RegisterWSConnection(ctx, userID, connID, newPresence)
		return err
	}

	return s.cache.Heartbeat(ctx, userID, connID)
}
func (s *Service) pubUpdatePresence(ctx context.Context, userID fields.ID, p user.Presence) {
	nodeToUsers, err := s.cache.GetUpdateRecipientNodes(ctx, userID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get update recipient nodes", "user_id", userID, "error", err)
		return
	}

	if len(nodeToUsers) == 0 {
		return
	}

	payload := user.EventUpdatePresencePayload{
		UserID:   userID.String(),
		Presence: p.String(),
	}

	if err := s.publishToNodes(ctx, nodeToUsers, user.EventUpdatePresence, payload); err != nil {
		slog.ErrorContext(ctx, "failed to broadcast presence updates to nodes",
			"user_id", userID,
			"error", err,
		)
	}
}

func (s *Service) publishToNodes(
	ctx context.Context,
	nodeToUsers map[fields.ID][]fields.ID,
	eventType string,
	payload interface{},
) error {
	if len(nodeToUsers) == 0 {
		return nil
	}

	rawPayload, err := json.Marshal(payload)
	if err != nil {
		return errs.Internal("Failed to marshal event payload.").Wrap(err)
	}

	nodeEvents := make(map[fields.ID]NodeEvent, len(nodeToUsers))
	for nodeID, targetUserIDs := range nodeToUsers {
		nodeEvents[nodeID] = NodeEvent{
			UserIDs: fields.UUIDs(targetUserIDs),
			Type:    eventType,
			Data:    rawPayload,
		}
	}

	return s.pub.PublishBatchNodeEvents(ctx, nodeEvents)
}
