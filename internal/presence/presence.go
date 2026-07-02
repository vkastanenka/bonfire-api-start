package presence

import "bonfire-api/internal/redis"

// --- presence types ---

type Activity = redis.ActivityStatus

const (
	StatusOnline    = redis.StatusOnline
	StatusBusy      = redis.StatusBusy
	StatusDND       = redis.StatusDND
	StatusInvisible = redis.StatusInvisible
	StatusOffline   = redis.StatusOffline
)
