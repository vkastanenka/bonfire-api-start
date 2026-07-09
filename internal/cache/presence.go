package cache

import (
	"context"
	"errors"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// Presence represents the computed, real-time visibility state of a user session.
type Presence string

const (
	PresenceOnline    Presence = "online"
	PresenceOffline   Presence = "offline"
	PresenceIdle      Presence = "idle"
	PresenceBusy      Presence = "busy"
	PresenceDND       Presence = "dnd"
	PresenceInvisible Presence = "invisible"
)

func (s Presence) Valid() bool {
	switch s {
	case PresenceOnline, PresenceIdle, PresenceBusy, PresenceDND, PresenceInvisible:
		return true
	}
	return false
}

const presenceTTL = 30 * time.Second

func (m *manager) Heartbeat(ctx context.Context, userID string) error {
	key := PresenceUserKey(userID)
	return m.client.Set(ctx, key, "1", presenceTTL).Err()
}

func (m *manager) SetStatus(ctx context.Context, userID string, status Presence) error {
	if !status.Valid() {
		return fmt.Errorf("invalid activity status")
	}
	return m.client.Set(ctx, StatusUserKey(userID), string(status), 0).Err()
}

func (m *manager) GetStatus(ctx context.Context, userID string) (Presence, error) {
	value, err := m.client.Get(ctx, StatusUserKey(userID)).Result()
	if errors.Is(err, goredis.Nil) {
		return PresenceOnline, nil
	}
	if err != nil {
		return "", err
	}
	return Presence(value), nil
}

// GetActivity evaluates the intersection of an active heartbeat and their visibility selection.
// Fixed: Named to match PresenceTracker interface and captured the 'online' boolean state.
func (m *manager) GetActivity(ctx context.Context, userID string) (Presence, error) {
	online, err := m.getPresence(ctx, userID)
	if err != nil {
		return "", err
	}

	status, err := m.GetStatus(ctx, userID)
	if err != nil {
		return "", err
	}

	if status == PresenceInvisible || !online {
		return PresenceOffline, nil
	}

	return status, nil
}

// GetBulkActivity uses an optimized MGet pipeline to compile real-time states.
// Fixed: Completed implementation, resolved typos, and fulfilled interface contract.
func (m *manager) GetBulkActivity(ctx context.Context, userIDs []string) (map[string]Presence, error) {
	if len(userIDs) == 0 {
		return map[string]Presence{}, nil
	}

	presenceKeys := make([]string, len(userIDs))
	statusKeys := make([]string, len(userIDs))

	for i, id := range userIDs {
		presenceKeys[i] = PresenceUserKey(id)
		statusKeys[i] = StatusUserKey(id)
	}

	// Fetch all heartbeats sequentially via atomic batching
	presences, err := m.client.MGet(ctx, presenceKeys...).Result()
	if err != nil {
		return nil, err
	}

	// Fetch explicit custom statuses
	statuses, err := m.client.MGet(ctx, statusKeys...).Result()
	if err != nil {
		return nil, err
	}

	activities := make(map[string]Presence, len(userIDs))

	for i, id := range userIDs {
		online := presences[i] != nil
		status := PresenceOnline // Default fallback state

		if statuses[i] != nil {
			if value, ok := statuses[i].(string); ok {
				s := Presence(value)
				if s.Valid() {
					status = s
				}
			}
		}

		switch {
		case status == PresenceInvisible || !online:
			activities[id] = PresenceOffline
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
	if errors.Is(err, goredis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
