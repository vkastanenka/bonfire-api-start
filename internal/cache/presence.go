package cache

import (
	"context"
	"errors"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
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
	case StatusOnline, StatusBusy, StatusDND, StatusInvisible:
		return true
	}
	return false
}

const presenceTTL = 30 * time.Second

func (m *manager) Heartbeat(ctx context.Context, userID string) error {
	key := PresenceUserKey(userID)
	return m.client.Set(ctx, key, "1", presenceTTL).Err()
}

func (m *manager) SetStatus(ctx context.Context, userID string, status ActivityStatus) error {
	if !status.Valid() {
		return fmt.Errorf("invalid activity status")
	}
	return m.client.Set(ctx, StatusUserKey(userID), string(status), 0).Err()
}

func (m *manager) GetStatus(ctx context.Context, userID string) (ActivityStatus, error) {
	value, err := m.client.Get(ctx, StatusUserKey(userID)).Result()
	if errors.Is(err, goredis.Nil) { // Fixed: Using explicit goredis alias
		return StatusOnline, nil
	}
	if err != nil {
		return "", err
	}
	return ActivityStatus(value), nil
}

func (m *manager) GetActivity(ctx context.Context, userID string) (ActivityStatus, error) {
	online, err := m.getPresence(ctx, userID)
	if err != nil {
		return "", err
	}

	status, err := m.GetStatus(ctx, userID)
	if err != nil {
		return "", err
	}

	if status == StatusInvisible || !online {
		return StatusOffline, nil
	}

	return status, nil
}

func (m *manager) GetBulkActivity(ctx context.Context, userIDs []string) (map[string]ActivityStatus, error) {
	if len(userIDs) == 0 {
		return map[string]ActivityStatus{}, nil
	}

	presenceKeys := make([]string, len(userIDs))
	statusKeys := make([]string, len(userIDs))

	for i, id := range userIDs {
		presenceKeys[i] = PresenceUserKey(id)
		statusKeys[i] = StatusUserKey(id)
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
		case status == StatusInvisible || !online:
			activities[id] = StatusOffline
		default:
			activities[id] = status
		}
	}

	return activities, nil
}

// unexported helper method for local checks
func (m *manager) getPresence(ctx context.Context, userID string) (bool, error) {
	key := PresenceUserKey(userID)
	err := m.client.Get(ctx, key).Err()
	if errors.Is(err, goredis.Nil) { // Fixed: Using explicit goredis alias
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
