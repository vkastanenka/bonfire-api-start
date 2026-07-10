package cache

import (
	"github.com/redis/go-redis/v9"
)

type client struct {
	redis *redis.Client
}

func New(rClient *redis.Client) Client {
	return &client{
		redis: rClient,
	}
}
