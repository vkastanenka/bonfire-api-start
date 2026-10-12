package session

import (
	"time"

	"github.com/google/uuid"
)

const (
	CacheTTL    = 24 * time.Hour
	minCacheTTL = 1 * time.Second
)

func CacheKey(id uuid.UUID) string { return "{session:" + id.String() + "}" }

func CalcCacheTTL(sess *Session) (time.Duration, bool) {
	if sess.ExpiresAt.IsZero() {
		return CacheTTL, true
	}

	remaining := time.Until(sess.ExpiresAt)
	if remaining < minCacheTTL {
		return 0, false
	}

	return remaining, true
}
