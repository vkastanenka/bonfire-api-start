package gateway

import (
	"bonfire-api/internal/pkg/errs"
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

type Broadcaster struct {
	cache  *Cache
	pubsub pubSub
}

func NewBroadcaster(cache *Cache, pubsub pubSub) *Broadcaster {
	return &Broadcaster{
		cache:  cache,
		pubsub: pubsub,
	}
}

// ToUser broadcasts an event to a single target user with optional session exclusions.
func (b *Broadcaster) ToUser(
	ctx context.Context,
	recipientID uuid.UUID,
	messageType string,
	payload any,
	excludeSessionIDs ...uuid.UUID,
) error {
	return b.publish(ctx, []uuid.UUID{recipientID}, nil, excludeSessionIDs, messageType, payload)
}

// ToUsers broadcasts an event to multiple target users with optional session exclusions.
func (b *Broadcaster) ToUsers(
	ctx context.Context,
	recipientIDs []uuid.UUID,
	messageType string,
	payload any,
	excludeSessionIDs ...uuid.UUID,
) error {
	return b.publish(ctx, recipientIDs, nil, excludeSessionIDs, messageType, payload)
}

// ToUserSession sends an event to a single specific session belonging to a user.
func (b *Broadcaster) ToUserSession(
	ctx context.Context,
	userID, sessionID uuid.UUID,
	messageType string,
	payload any,
) error {
	return b.ToUserSessions(ctx, userID, []uuid.UUID{sessionID}, messageType, payload)
}

// ToUserSessions sends targeted events to specific sessions belonging to a single user.
func (b *Broadcaster) ToUserSessions(
	ctx context.Context,
	userID uuid.UUID,
	sessionIDs []uuid.UUID,
	messageType string,
	payload any,
) error {
	if len(sessionIDs) == 0 {
		return nil
	}

	userSessions := map[uuid.UUID][]uuid.UUID{
		userID: sessionIDs,
	}

	return b.publish(ctx, []uuid.UUID{userID}, userSessions, nil, messageType, payload)
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

	var excludeSet map[uuid.UUID]bool
	if len(excludeSessionIDs) > 0 {
		excludeSet = make(map[uuid.UUID]bool, len(excludeSessionIDs))
		for _, sID := range excludeSessionIDs {
			if sID != uuid.Nil {
				excludeSet[sID] = true
			}
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
	hubIDsMap := make(map[string]uuid.UUID)

	for userID, fields := range userHashes {
		userTargetSessions := targetSessionMap[userID]

		for field, hubIDStr := range fields {
			if field == "presence" {
				continue
			}

			sessionID, err := uuid.Parse(field)
			if err != nil {
				continue
			}

			if hasTargetSessions {
				if !userTargetSessions[sessionID] {
					continue
				}
			} else {
				if excludeSet[sessionID] {
					continue
				}
			}

			hubID, cached := hubIDsMap[hubIDStr]
			if !cached {
				hubID, err = uuid.Parse(hubIDStr)
				if err != nil {
					continue
				}
				hubIDsMap[hubIDStr] = hubID
			}

			hubTargets[hubID] = append(hubTargets[hubID], sessionID)
		}
	}

	if len(hubTargets) == 0 {
		return nil
	}

	hubEvents := make(map[uuid.UUID]Event, len(hubTargets))
	for hubID, sessionIDs := range hubTargets {
		hubEvents[hubID] = Event{
			SessionIDs: sessionIDs,
			Message:    messageBytes,
		}
	}

	return b.publishToHubs(ctx, hubEvents)
}

func (b *Broadcaster) publishToHubs(ctx context.Context, hubEvents map[uuid.UUID]Event) error {
	if len(hubEvents) == 0 {
		return nil
	}

	pubMessages := make(map[string]any, len(hubEvents))
	for hubID, event := range hubEvents {
		pubMessages[EventsPubSubChannel(hubID)] = event
	}

	if err := b.pubsub.PublishBatch(ctx, pubMessages); err != nil {
		return err
	}

	return nil
}
