package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Querier interface {
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Get(ctx context.Context, key string, dest interface{}) error
	MGet(ctx context.Context, keys ...string) ([]interface{}, error)
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	Increment(ctx context.Context, key string, ttl time.Duration) (int64, error)
	HSet(ctx context.Context, key string, field string, value interface{}) error
	HDel(ctx context.Context, key string, fields ...string) error
	HGetAll(ctx context.Context, key string, dest *map[string]string) error
	Expire(ctx context.Context, key string, ttl time.Duration) error
	Publish(ctx context.Context, channel string, payload interface{}) error
}

var _ Querier = (*Queries)(nil)

type Store interface {
	Querier
	ExecPipeline(ctx context.Context, fn func(Querier) error) error
}

type Manager interface {
	Store
	MessageBus
}

type manager struct {
	*Queries
	client *redis.Client
}

func NewManager(client *redis.Client) Manager {
	return &manager{
		Queries: New(client),
		client:  client,
	}
}

func (m *manager) ExecPipeline(ctx context.Context, fn func(Querier) error) error {
	pipe := m.client.Pipeline()
	qpipe := m.WithPipeline(pipe)

	if err := fn(qpipe); err != nil {
		pipe.Discard()
		return fmt.Errorf("pipeline queue failed: %w", err)
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return NewError(err, ScopeStore)
	}

	return nil
}

func (m *manager) Subscribe(ctx context.Context, channel string) (Subscription, error) {
	pb := m.client.Subscribe(ctx, channel)

	subCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if _, err := pb.Receive(subCtx); err != nil {
		pb.Close()
		return nil, NewError(err, ScopeEvents)
	}

	sub := &cacheSub{
		pubsub: pb,
		ch:     make(chan string, defaultChannelBuffer),
		done:   make(chan struct{}),
	}

	go sub.listen()
	return sub, nil
}
