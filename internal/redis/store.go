package redis

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type Store struct {
	*Queries
	client *redis.Client
	scope  Scope
}

func NewStore(client *redis.Client) *Store {
	return &Store{
		Queries: New(client),
		client:  client,
		scope:   ScopeStore,
	}
}

func (s *Store) WithScope(scope Scope) *Store {
	return &Store{
		Queries: s.Queries,
		client:  s.client,
		scope:   scope,
	}
}

func (s *Store) Scope() Scope {
	return s.scope
}

func (s *Store) Err(err error) error {
	return NewError(err, s.scope)
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
		return s.Err(err)
	}

	return nil
}
