package redis

import (
	"context"
	"errors"
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

func (s *Store) ExecPipelineFunc(ctx context.Context, fn func(pipe redis.Pipeliner) error) error {
	pipe := s.client.Pipeline()

	if err := fn(pipe); err != nil {
		pipe.Discard()
		return fmt.Errorf("failed to enqueue pipeline commands: %w", err)
	}

	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		return NewError(err, ScopeStore)
	}

	return nil
}
