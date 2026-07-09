package cache

import (
	"bonfire-api/internal/apperr"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Presence uint8

const (
	PresenceUnknown   Presence = iota // 0
	PresenceOnline                    // 1
	PresenceOffline                   // 2
	PresenceIdle                      // 3
	PresenceBusy                      // 4
	PresenceDND                       // 5
	PresenceInvisible                 // 6
	presenceMax                       // 7
)

// TODO: Move to config
const presenceTTL = 30 * time.Second

func (s Presence) Valid() bool {
	return s > PresenceUnknown && s < presenceMax
}

func (s Presence) String() string {
	switch s {
	case PresenceOnline:
		return "online"
	case PresenceIdle:
		return "idle"
	case PresenceBusy:
		return "busy"
	case PresenceDND:
		return "dnd"
	default:
		return "offline"
	}
}

func ParsePresence(s string) Presence {
	switch s {
	case "online":
		return PresenceOnline
	case "idle":
		return PresenceIdle
	case "busy":
		return PresenceBusy
	case "dnd":
		return PresenceDND
	default:
		return PresenceOffline
	}
}

func (m *manager) Heartbeat(ctx context.Context, userID uuid.UUID, p Presence) error {
	if !p.Valid() {
		return apperr.NewInvalidInput(nil, fmt.Sprintf("invalid presence: %d", p))
	}
	key := UserPresenceKey(userID)
	return NewError(m.client.Set(ctx, key, p.String(), presenceTTL).Err(), ScopePresence)
}

func (m *manager) GetPresence(ctx context.Context, userID uuid.UUID) (Presence, error) {
	value, err := m.client.Get(ctx, UserPresenceKey(userID)).Result()

	if IsNotFoundError(err) {
		return PresenceOffline, nil
	}

	if err != nil {
		return PresenceUnknown, NewError(err, ScopePresence)
	}

	return ParsePresence(value), nil
}

func (m *manager) GetBulkPresence(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]Presence, error) {
	if len(userIDs) == 0 {
		return map[uuid.UUID]Presence{}, nil
	}

	presenceKeys := make([]string, len(userIDs))
	for i, id := range userIDs {
		presenceKeys[i] = UserPresenceKey(id)
	}

	values, err := m.client.MGet(ctx, presenceKeys...).Result()
	if err != nil {
		return nil, NewError(err, ScopePresence)
	}

	activities := make(map[uuid.UUID]Presence, len(userIDs))

	for i, id := range userIDs {
		if values[i] == nil {
			activities[id] = PresenceOffline
			continue
		}

		if valStr, ok := values[i].(string); ok {
			activities[id] = ParsePresence(valStr)
		} else {
			activities[id] = PresenceOffline
		}
	}

	return activities, nil
}
