package user

import (
	"time"

	"github.com/google/uuid"
)

const (
	CacheTTL = 24 * time.Hour
)

func CacheKey(id uuid.UUID) string { return "{user:" + id.String() + "}" }
