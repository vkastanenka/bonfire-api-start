// internal/redis/manager.go
package redis

import (
	goredis "github.com/redis/go-redis/v9"
)

// manager is private to prevent external instantiation without our constructor.
type manager struct {
	client *goredis.Client
}

// NewManager binds a live connection pool to our application capability layers.
func NewManager(client *goredis.Client) Manager {
	return &manager{
		client: client,
	}
}
