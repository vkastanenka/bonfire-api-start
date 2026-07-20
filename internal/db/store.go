package db

import (
	"context"
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

func (s *store) ExecTx(ctx context.Context, fn func(Querier) error) error {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	qtx := s.WithTx(tx)

	err = fn(qtx)
	if err != nil {
		rollbackCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		rbErr := tx.Rollback(rollbackCtx)
		cancel()

		if rbErr != nil {
			slog.Error("transaction rollback failed entirely", "original_error", err, "rollback_error", rbErr)
			return fmt.Errorf("tx error: %w, rollback error: %v", err, rbErr)
		}

		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}
