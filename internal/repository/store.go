package repository

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TODO: Implement after other refactor
// type Store interface {
// 	Querier // This includes all your generated sqlc methods
// 	ExecTx(ctx context.Context, fn func(*Queries) error) error
// }

// SQLStore provides a repository implementation
type SQLStore struct {
	db *pgxpool.Pool
	*Queries
}

// NewStore initializes a new SQLStore with a database connection pool and query interface
func NewStore(db *pgxpool.Pool) *SQLStore {
	return &SQLStore{
		db:      db,
		Queries: New(db),
	}
}

// ExecTx executes operations inside a transaction block.
func (s *SQLStore) ExecTx(ctx context.Context, fn func(*Queries) error) error {
	// Begin transaction
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		slog.Error("failed to begin transaction", "error", err)
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Route queries through transaction
	qtx := s.WithTx(tx)

	// Pass transaction queries to callback function
	err = fn(qtx)
	if err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			slog.Error("tx rollback failed", "original_error", err, "rollback_error", rbErr)
			return fmt.Errorf("tx error: %v, rollback error: %v", err, rbErr)
		}
		return err
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		slog.Error("failed to commit transaction", "error", err)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}
