package cache

import (
	"github.com/redis/go-redis/v9"
)

type manager struct {
	client *redis.Client
}

func NewManager(client *redis.Client) Manager {
	return &manager{
		client: client,
	}
}
