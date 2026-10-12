package cache


import (
    "context"
    "time"
    
    redisdriver "github.com/redis/go-redis/v9"
)

type Client struct {
    rdb redisdriver.Cmdable
}

func NewClient(rdb redisdriver.Cmdable) *Client {
    return &Client{rdb: rdb}
}

// Low-level primitives that handle all marshaling, unmarshaling, and pipelining app-wide
func (c *Client) Get(ctx context.Context, key string, dest any) error { ... }
func (c *Client) Set(ctx context.Context, key string, val any, ttl time.Duration) error { ... }
func (c *Client) Del(ctx context.Context, keys ...string) error { ... }

// Pipelining is exposed natively and cleanly app-wide
func (c *Client) Pipelined(ctx context.Context, fn func(pipe redisdriver.Pipeliner) error) error {
    _, err := c.rdb.Pipelined(ctx, fn)
    return err
}