package cache

import (
	"context"
	"fmt"

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
