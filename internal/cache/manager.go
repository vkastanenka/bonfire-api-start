package cache

import (
	goredis "github.com/redis/go-redis/v9"
)

type manager struct {
	client *goredis.Client
}

func NewManager(client *goredis.Client) Manager {
	return &manager{
		client: client,
	}
}
