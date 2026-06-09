package repository

import (
	"context"
	"fmt"

	"bonfire-api/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Ensure PostgresRepository strictly implements domain.DBRepository at compile time
var _ domain.DBRepository = (*PostgresRepository)(nil)

type PostgresRepository struct {
	pool    *pgxpool.Pool
	queries *Queries // The global, non-transactional base queries instance from sqlc
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{
		pool:    pool,
		queries: New(pool), // sqlc constructor
	}
}

// WithTx manages the lifecycle of a transaction and injects sqlc queries into the context scope.
func (r *PostgresRepository) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	// 1. Begin the database transaction
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}

	// 2. Ensure rollback runs with a safe, uncancelled context if a panic or error occurs
	defer tx.Rollback(context.WithoutCancel(ctx))

	// 3. Bind our sqlc Queries instance to this specific transaction
	txQueries := r.queries.WithTx(tx)

	// 4. Inject the transactional queries into the context chain
	txCtx := InjectQueries(ctx, txQueries)

	// 5. Execute the business logic function using our new context
	if err := fn(txCtx); err != nil {
		return err // Implicitly triggers the deferred Rollback
	}

	// 6. Commit cleanly using a detached context to survive client network hangups
	if err := tx.Commit(context.WithoutCancel(ctx)); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	return nil
}
