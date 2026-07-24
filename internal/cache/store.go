package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Store struct {
	*Queries
	client *redis.Client
}

func NewStore(client *redis.Client) *Store {
	return &Store{
		Queries: New(client),
		client:  client,
	}
}

func (s *Store) ExecPipeline(ctx context.Context, fn func(Querier) error) error {
	pipe := s.client.Pipeline()
	qpipe := s.WithPipeline(pipe)

	if err := fn(qpipe); err != nil {
		pipe.Discard()
		return fmt.Errorf("pipeline queue failed: %w", err)
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return NewError(err, ScopeStore)
	}

	return nil
}

func (s *Store) Subscribe(ctx context.Context, channel string) (Subscription, error) {
	pb := s.client.Subscribe(ctx, channel)

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

func (s *Store) PSubscribe(ctx context.Context, patterns ...string) (Subscription, error) {
	pb := s.client.PSubscribe(ctx, patterns...)

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
