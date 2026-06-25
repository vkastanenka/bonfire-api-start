package cache

import (
	"errors"

	"github.com/redis/go-redis/v9"
)

// This struct holds your raw client and implements the methods for all three interfaces.

// ErrCacheMiss is returned when a key is not found, standardizing the error across the app.
var ErrCacheMiss = errors.New("cache: key not found")

type redisManager struct {
	client *redis.Client
}

// NewManager creates our application-level Redis wrapper.
func NewManager(client *redis.Client) Manager {
	return &redisManager{
		client: client,
	}
}
