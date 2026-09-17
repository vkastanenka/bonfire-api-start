package db

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"bonfire-api/internal/pkg/errs"

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
	cdb := newContextDB(pool)
	return &Store{
		Queries: New(cdb),
		pool:    pool,
	}
}

// ExecTx executes fn within a database transaction, reusing an existing in-context
// transaction or managing a new transaction lifecycle with rollback on error or panic.
func (s *Store) ExecTx(ctx context.Context, fn func(txCtx context.Context) error) (err error) {
	if _, ok := ExtractTx(ctx); ok {
		return fn(ctx)
	}

	tx, beginErr := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if beginErr != nil {
		return errs.Internal("Failed to begin database transaction.").
			Reason("DB_TX_BEGIN_FAILED").
			Wrap(beginErr)
	}

	defer func() {
		if p := recover(); p != nil {
			rbCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			_ = tx.Rollback(rbCtx)
			cancel()
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
				err = fmt.Errorf("tx error: %w (rollback failed: %v)", err, rbErr)
			}
		}
	}()

	txCtx := InjectTx(ctx, tx)

	if err = fn(txCtx); err != nil {
		return err
	}

	if commitErr := tx.Commit(ctx); commitErr != nil {
		return errs.Internal("Failed to commit database transaction.").
			Reason("DB_TX_COMMIT_FAILED").
			Wrap(commitErr)
	}

	return nil
}
