package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type ActivityStatus string

const (
	StatusOnline    ActivityStatus = "online"
	StatusBusy      ActivityStatus = "busy"
	StatusDND       ActivityStatus = "dnd"
	StatusInvisible ActivityStatus = "invisible"
	StatusOffline   ActivityStatus = "offline"
)

func (s ActivityStatus) Valid() bool {
	switch s {
	case StatusOnline,
		StatusBusy,
		StatusDND,
		StatusInvisible:
		return true
	}

	return false
}

// Pillar 2: Heartbeats
// This is heavily optimized for a chat application.
// Notice the GetBulkPresence function: in Discord, when you load a server, you need to know the online status of 1,000 members immediately.
// We use Redis MGET to fetch them all in a single lightning-fast query.

// presenceTTL is how long a user remains "Online" without sending a heartbeat.
const presenceTTL = 30 * time.Second

func (m *redisManager) Heartbeat(ctx context.Context, userID string) error {
	key := UserPresenceKey(userID)
	// Sets the value to "1" and resets the 30-second expiration timer
	return m.client.Set(ctx, key, "1", presenceTTL).Err()
}

func (m *redisManager) SetStatus(
	ctx context.Context,
	userID string,
	status ActivityStatus,
) error {
	if !status.Valid() {
		return fmt.Errorf("invalid activity status")
	}

	return m.client.Set(
		ctx,
		UserStatusKey(userID),
		status,
		0,
	).Err()
}

func (m *redisManager) GetStatus(
	ctx context.Context,
	userID string,
) (ActivityStatus, error) {

	value, err := m.client.Get(
		ctx,
		UserStatusKey(userID),
	).Result()

	if err == redis.Nil {
		return StatusOnline, nil
	}

	if err != nil {
		return "", err
	}

	return ActivityStatus(value), nil
}

func (m *redisManager) GetActivity(
	ctx context.Context,
	userID string,
) (ActivityStatus, error) {

	online, err := m.GetPresence(ctx, userID)
	if err != nil {
		return "", err
	}

	status, err := m.GetStatus(ctx, userID)
	if err != nil {
		return "", err
	}

	if status == StatusInvisible {
		return StatusOffline, nil
	}

	if !online {
		return StatusOffline, nil
	}

	return status, nil
}

func (m *redisManager) GetBulkActivity(
	ctx context.Context,
	userIDs []string,
) (map[string]ActivityStatus, error) {
	if len(userIDs) == 0 {
		return map[string]ActivityStatus{}, nil
	}

	presenceKeys := make([]string, len(userIDs))
	statusKeys := make([]string, len(userIDs))

	for i, id := range userIDs {
		presenceKeys[i] = UserPresenceKey(id)
		statusKeys[i] = UserStatusKey(id)
	}

	presences, err := m.client.MGet(ctx, presenceKeys...).Result()
	if err != nil {
		return nil, err
	}

	statuses, err := m.client.MGet(ctx, statusKeys...).Result()
	if err != nil {
		return nil, err
	}

	activities := make(map[string]ActivityStatus, len(userIDs))

	for i, id := range userIDs {
		online := presences[i] != nil

		// Default status if none has been explicitly set.
		status := StatusOnline

		if statuses[i] != nil {
			if value, ok := statuses[i].(string); ok {
				s := ActivityStatus(value)
				if s.Valid() {
					status = s
				}
			}
		}

		switch {
		case status == StatusInvisible:
			activities[id] = StatusOffline

		case !online:
			activities[id] = StatusOffline

		default:
			activities[id] = status
		}
	}

	return activities, nil
}

func (m *redisManager) GetPresence(ctx context.Context, userID string) (bool, error) {
	key := UserPresenceKey(userID)
	err := m.client.Get(ctx, key).Err()
	if err == redis.Nil {
		return false, nil // Key expired, user is offline
	}
	if err != nil {
		return false, err
	}
	return true, nil // Key exists, user is online
}
