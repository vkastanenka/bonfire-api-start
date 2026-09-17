package db

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type txKey struct{}

// InjectTx attaches an active pgx.Tx to the provided context.
func InjectTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}

// ExtractTx retrieves the active pgx.Tx from the context if present.
func ExtractTx(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(txKey{}).(pgx.Tx)
	return tx, ok
}

// IsTx checks whether the context currently carries an active transaction.
func IsTx(ctx context.Context) bool {
	_, ok := ExtractTx(ctx)
	return ok
}

// ctxDB wraps a pgxpool.Pool and dynamically routes operations to an in-context transaction or the pool.
type ctxDB struct {
	pool *pgxpool.Pool
}

// newContextDB returns a DBTX interface that auto-detects active transactions in the context.
func newContextDB(pool *pgxpool.Pool) DBTX {
	return &ctxDB{pool: pool}
}

// Exec executes a command on the in-context transaction, falling back to the pool.
func (c *ctxDB) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	if tx, ok := ExtractTx(ctx); ok {
		return tx.Exec(ctx, sql, arguments...)
	}
	return c.pool.Exec(ctx, sql, arguments...)
}

// Query executes a multi-row query on the in-context transaction, falling back to the pool.
func (c *ctxDB) Query(ctx context.Context, sql string, arguments ...any) (pgx.Rows, error) {
	if tx, ok := ExtractTx(ctx); ok {
		return tx.Query(ctx, sql, arguments...)
	}
	return c.pool.Query(ctx, sql, arguments...)
}

// QueryRow executes a single-row query on the in-context transaction, falling back to the pool.
func (c *ctxDB) QueryRow(ctx context.Context, sql string, arguments ...any) pgx.Row {
	if tx, ok := ExtractTx(ctx); ok {
		return tx.QueryRow(ctx, sql, arguments...)
	}
	return c.pool.QueryRow(ctx, sql, arguments...)
}

// CopyFrom performs a bulk copy operation on the in-context transaction, falling back to the pool.
func (c *ctxDB) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	if tx, ok := ExtractTx(ctx); ok {
		return tx.CopyFrom(ctx, tableName, columnNames, rowSrc)
	}
	return c.pool.CopyFrom(ctx, tableName, columnNames, rowSrc)
}
