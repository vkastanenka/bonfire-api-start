package db

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	*Queries
	pool   *pgxpool.Pool
	entity Entity
}

func NewStore(pool *pgxpool.Pool) *Store {
	cdb := newContextDB(pool)
	return &Store{
		Queries: New(cdb),
		pool:    pool,
		entity:  Entity("store"),
	}
}

func (s *Store) WithEntity(entity Entity) *Store {
	return &Store{
		Queries: s.Queries,
		pool:    s.pool,
		entity:  entity,
	}
}

func (s *Store) Entity() Entity {
	return s.entity
}

func (s *Store) Err(err error) error {
	return NewError(err, s.entity)
}

func (s *Store) ExecTx(ctx context.Context, fn func(txCtx context.Context) error) error {
	// 1. If we are ALREADY inside a transaction, reuse it directly.
	if _, ok := ExtractTx(ctx); ok {
		return fn(ctx)
	}

	// 2. Outer-most call: Begin a real database transaction.
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return s.Err(err)
	}

	defer func() {
		if err != nil {
			rbCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			rbErr := tx.Rollback(rbCtx)
			if rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) {
				if !errors.Is(ctx.Err(), context.Canceled) {
					slog.Error("transaction rollback failed", "original_error", err, "rollback_error", rbErr)
				}
				err = fmt.Errorf("tx error: %w (rollback failed: %v)", err, rbErr)
			}
		}
	}()

	txCtx := InjectTx(ctx, tx)

	if err = fn(txCtx); err != nil {
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		return s.Err(err)
	}

	return nil
}
