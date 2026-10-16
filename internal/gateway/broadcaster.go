package gateway

import (
	"bonfire-api/internal/pkg/errs"
	"bonfire-api/internal/redis"
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	redisdriver "github.com/redis/go-redis/v9"
)

type Broadcaster struct {
	rdb   redisdriver.Cmdable
	cache Cache
}

func NewBroadcaster(rdb redisdriver.Cmdable, cache Cache) *Broadcaster {
	return &Broadcaster{
		rdb:   rdb,
		cache: cache,
	}
}

// BroadcastToUser broadcasts an event to a single target user with optional session exclusions.
func (b *Broadcaster) BroadcastToUser(
	ctx context.Context,
	recipientID uuid.UUID,
	messageType string,
	payload any,
	excludeSessionIDs ...uuid.UUID,
) error {
	return b.publish(ctx, []uuid.UUID{recipientID}, nil, excludeSessionIDs, messageType, payload)
}

// BroadcastToUsers broadcasts an event to multiple target users with optional session exclusions.
func (b *Broadcaster) BroadcastToUsers(
	ctx context.Context,
	recipientIDs []uuid.UUID,
	messageType string,
	payload any,
	excludeSessionIDs ...uuid.UUID,
) error {
	return b.publish(ctx, recipientIDs, nil, excludeSessionIDs, messageType, payload)
}

// BroadcastToUserSession sends an event to a single specific session belonging to a user.
func (b *Broadcaster) BroadcastToUserSession(
	ctx context.Context,
	userID, sessionID uuid.UUID,
	messageType string,
	payload any,
) error {
	userSessions := map[uuid.UUID][]uuid.UUID{
		userID: {sessionID},
	}
	return b.BroadcastToUserSessions(ctx, userSessions, messageType, payload)
}

// BroadcastToUserSessions sends targeted events to specific sessions grouped by user (map[userID][]sessionIDs).
func (b *Broadcaster) BroadcastToUserSessions(
	ctx context.Context,
	userSessions map[uuid.UUID][]uuid.UUID,
	messageType string,
	payload any,
	excludeSessionIDs ...uuid.UUID,
) error {
	if len(userSessions) == 0 {
		return nil
	}

	userIDs := make([]uuid.UUID, 0, len(userSessions))
	for userID := range userSessions {
		userIDs = append(userIDs, userID)
	}

	return b.publish(ctx, userIDs, userSessions, excludeSessionIDs, messageType, payload)
}

func (b *Broadcaster) publish(
	ctx context.Context,
	userIDs []uuid.UUID,
	targetSessions map[uuid.UUID][]uuid.UUID,
	excludeSessionIDs []uuid.UUID,
	messageType string,
	payload any,
) error {
	if len(userIDs) == 0 {
		return nil
	}

	userHashes, err := b.cache.GetBatchUsers(ctx, userIDs)
	if err != nil {
		return err
	}

	if len(userHashes) == 0 {
		return nil
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return errs.Internal("Failed to marshal event payload.").Wrap(err)
	}

	messageBytes, err := json.Marshal(Message{
		Type: messageType,
		Data: payloadBytes,
	})
	if err != nil {
		return errs.Internal("Failed to marshal WS message.").Wrap(err)
	}

	excludeSet := make(map[uuid.UUID]bool, len(excludeSessionIDs))
	for _, sID := range excludeSessionIDs {
		if sID != uuid.Nil {
			excludeSet[sID] = true
		}
	}

	hasTargetSessions := len(targetSessions) > 0
	targetSessionMap := make(map[uuid.UUID]map[uuid.UUID]bool, len(targetSessions))
	if hasTargetSessions {
		for uID, sIDs := range targetSessions {
			sMap := make(map[uuid.UUID]bool, len(sIDs))
			for _, sID := range sIDs {
				if sID != uuid.Nil {
					sMap[sID] = true
				}
			}
			if len(sMap) > 0 {
				targetSessionMap[uID] = sMap
			}
		}
	}

	hubTargets := make(map[uuid.UUID][]uuid.UUID)

	for userID, fields := range userHashes {
		userTargetSessions := targetSessionMap[userID]

		for field, hubIDStr := range fields {
			// Skip the presence field
			if field == "presence" {
				continue
			}

			// Parse composite field: session:{sessionID}:client:{clientID}
			sessionID, clientID, err := parseSessionClientField(field)
			if err != nil {
				continue
			}

			// Apply filtering logic
			if hasTargetSessions {
				// If targeting specific sessions, the client's session must be explicitly included
				if !userTargetSessions[sessionID] {
					continue
				}
			} else {
				// Otherwise, check if this session is in the exclusion list
				if excludeSet[sessionID] {
					continue
				}
			}

			hubID, err := uuid.Parse(hubIDStr)
			if err != nil {
				continue
			}

			hubTargets[hubID] = append(hubTargets[hubID], clientID)
		}
	}

	if len(hubTargets) == 0 {
		return nil
	}

	hubEvents := make(map[uuid.UUID]HubEvent, len(hubTargets))
	for hubID, clientIDs := range hubTargets {
		hubEvents[hubID] = HubEvent{
			ClientIDs: clientIDs,
			Message:   messageBytes,
		}
	}

	return b.publishToHubs(ctx, hubEvents)
}

func parseSessionClientField(field string) (uuid.UUID, uuid.UUID, error) {
	// Expected format: <uuid>:<uuid> -> 36 + 1 + 36 = 73 characters
	if len(field) != 73 || field[36] != ':' {
		return uuid.Nil, uuid.Nil, fmt.Errorf("invalid field format")
	}

	sessionID, err := uuid.Parse(field[0:36])
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}

	clientID, err := uuid.Parse(field[37:73])
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}

	return sessionID, clientID, nil
}

func (b *Broadcaster) publishToHubs(ctx context.Context, hubEvents map[uuid.UUID]HubEvent) error {
	if len(hubEvents) == 0 {
		return nil
	}

	messages := make([]redis.Message, 0, len(hubEvents))

	for hubID, event := range hubEvents {
		eventBytes, err := json.Marshal(event)
		if err != nil {
			return errs.Internal("Failed to marshal broadcast event envelope.").
				Reason("BROADCAST_MARSHAL_FAILED").
				Wrap(err)
		}
		messages = append(messages, redis.Message{
			Channel: EventsPubSubChannel(hubID),
			Payload: string(eventBytes),
		})
	}

	_, err := b.rdb.Pipelined(ctx, func(pipe redisdriver.Pipeliner) error {
		for _, message := range messages {
			pipe.Publish(ctx, message.Channel, message.Payload)
		}
		return nil
	})
	if err != nil {
		return errs.Unavailable("Failed to publish broadcast events to gateway nodes.").
			Reason("PUBSUB_BATCH_PUBLISH_FAILED").
			Wrap(err)
	}

	return nil
}
