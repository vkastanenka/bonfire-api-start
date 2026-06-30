package presence

import "bonfire-api/internal/cache"

// --- presence types ---

type Activity = cache.ActivityStatus

const (
	StatusOnline    = cache.StatusOnline
	StatusBusy      = cache.StatusBusy
	StatusDND       = cache.StatusDND
	StatusInvisible = cache.StatusInvisible
	StatusOffline   = cache.StatusOffline
)
