package redis

import (
	"context"

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

// WithScope creates a copy of Store configured with the target domain Scope.
func (s *Store) WithScope(scope Scope) *Store {
	return &Store{
		Queries: s.Queries.WithScope(scope),
		client:  s.client,
	}
}

func (s *Store) ExecPipeline(ctx context.Context, fn func(pipeCtx context.Context) error) error {
	if IsPipelined(ctx) {
		return fn(ctx)
	}

	pipe := s.client.Pipeline()
	pipeCtx := InjectCmdable(ctx, pipe)

	if err := fn(pipeCtx); err != nil {
		pipe.Discard()
		return err
	}

	if _, err := pipe.Exec(ctx); err != nil {
		pipe.Discard()
		return s.err(err)
	}

	return nil
}
