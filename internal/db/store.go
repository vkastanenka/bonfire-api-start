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

type Store interface {
	Querier
	ExecTx(ctx context.Context, fn func(Querier) error) error
}

type store struct {
	*Queries
	db *pgxpool.Pool
}

func NewStore(db *pgxpool.Pool) Store {
	return &store{
		Queries: New(db),
		db:      db,
	}
}

func (s *store) ExecTx(ctx context.Context, fn func(q Querier) error) (err error) {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
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

	qtx := s.WithTx(tx)

	if err = fn(qtx); err != nil {
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
