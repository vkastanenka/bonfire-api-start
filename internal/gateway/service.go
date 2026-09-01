package gateway

import (
	"bonfire-api/internal/errs"
	"bonfire-api/internal/fields"
	"bonfire-api/internal/pubsub"
	"bonfire-api/internal/user"
	"context"
	"encoding/json"
	"log/slog"
)

type Service struct {
	repo       UserRepository
	cache      UserCache
	tx         TX
	gatewayPub GatewayPub
}

func NewService(
	repo UserRepository,
	cache UserCache,
	tx TX,
	gatewayPub GatewayPub,
) *Service {
	return &Service{
		repo:       repo,
		cache:      cache,
		tx:         tx,
		gatewayPub: gatewayPub,
	}
}

func (s *Service) RegisterWSConnection(ctx context.Context, userID, nodeID fields.ID, presence user.Presence) error {
	// 1. Fetch current presence state from cache
	currentPresence, err := s.cache.GetPresence(ctx, userID)
	if err != nil {
		currentPresence = user.NewPresenceOffline()
	}

	// 2. Resolve the desired presence: default to current if valid, else fallback to Online
	p := presence
	if !p.IsValid() {
		if currentPresence.IsValid() && !currentPresence.IsOffline() {
			p = currentPresence // Preserve existing status (e.g., Busy / DND) for new tabs
		} else {
			p = user.NewPresenceOnline()
		}
	}

	// 3. Register the connection node in cache
	if err := s.cache.RegisterWSConnection(ctx, userID, nodeID, p); err != nil {
		return err
	}

	// 4. Broadcast ONLY if user transitioned from offline OR changed status
	if currentPresence.IsOffline() || currentPresence != p {
		s.pubUpdatePresence(ctx, userID, p)
	}

	return nil
}

func (s *Service) UnregisterWSConnection(ctx context.Context, userID, nodeID fields.ID) error {
	wentOffline, err := s.cache.UnregisterWSConnection(ctx, userID, nodeID)
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

func (s *Service) HandleHeartbeat(ctx context.Context, userID, nodeID fields.ID, newPresence user.Presence) error {
	currentPresence, err := s.cache.GetPresence(ctx, userID)
	if err != nil {
		return s.RegisterWSConnection(ctx, userID, nodeID, newPresence)
	}

	p := newPresence
	if !p.IsValid() {
		p = currentPresence
	}

	if currentPresence != p {
		return s.RegisterWSConnection(ctx, userID, nodeID, p)
	}

	return s.cache.Heartbeat(ctx, userID, nodeID)
}

func (s *Service) pubUpdatePresence(ctx context.Context, userID fields.ID, p user.Presence) {
	nodeToUsers, err := s.cache.GetUpdateRecipientNodes(ctx, userID)
	if err != nil || len(nodeToUsers) == 0 {
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

	nodeEvents := make(map[fields.ID]pubsub.NodeEvent, len(nodeToUsers))
	for nodeID, targetUserIDs := range nodeToUsers {
		nodeEvents[nodeID] = pubsub.NodeEvent{
			UserIDs: fields.UUIDs(targetUserIDs),
			Type:    eventType,
			Data:    rawPayload,
		}
	}

	return s.gatewayPub.PublishBatchNodeEvents(ctx, nodeEvents)
}
