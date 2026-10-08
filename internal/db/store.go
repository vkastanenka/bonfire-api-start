package db

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store provides database access methods and transaction management.
type Store struct {
	*Queries
	pool *pgxpool.Pool
}

// NewStore initializes a Store configured with dynamic transaction context routing.
func NewStore(pool *pgxpool.Pool) *Store {
	cdb := &ctxDB{pool: pool}
	return &Store{
		Queries: New(cdb),
		pool:    pool,
	}
}

// ExecTx executes fn within a tx, reusing a tx in ctx or instead beginning a new tx with rollback on error or panic.
func (s *Store) ExecTx(ctx context.Context, fn func(txCtx context.Context) error) (err error) {
	if _, ok := ExtractTx(ctx); ok {
		return fn(ctx)
	}

	tx, beginErr := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if beginErr != nil {
		return ErrDBTXStartFailed().Wrap(beginErr)
	}

	defer func() {
		if p := recover(); p != nil {
			rbCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			_ = tx.Rollback(rbCtx)
			panic(p)
		}

		if err != nil {
			rbCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			rbErr := tx.Rollback(rbCtx)
			if rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) {
				if !errors.Is(ctx.Err(), context.Canceled) {
					slog.ErrorContext(ctx, "transaction rollback failed",
						slog.Any("original_error", err),
						slog.Any("rollback_error", rbErr),
					)
				}
			}
		}
	}()

	txCtx := InjectTx(ctx, tx)

	if err = fn(txCtx); err != nil {
		return err
	}

	if commitErr := tx.Commit(ctx); commitErr != nil {
		return ErrDBTXCommitFailed().Wrap(commitErr)
	}

	return nil
}
