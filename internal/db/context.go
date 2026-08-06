// internal/db/conn_proxy.go
package db

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type txKey struct{}

func InjectTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}

func ExtractTx(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(txKey{}).(pgx.Tx)
	return tx, ok
}

type ctxDB struct {
	pool *pgxpool.Pool
}

func newContextDB(pool *pgxpool.Pool) DBTX {
	return &ctxDB{pool: pool}
}

func (c *ctxDB) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	if tx, ok := ExtractTx(ctx); ok {
		return tx.Exec(ctx, sql, arguments...)
	}
	return c.pool.Exec(ctx, sql, arguments...)
}

func (c *ctxDB) Query(ctx context.Context, sql string, arguments ...any) (pgx.Rows, error) {
	if tx, ok := ExtractTx(ctx); ok {
		return tx.Query(ctx, sql, arguments...)
	}
	return c.pool.Query(ctx, sql, arguments...)
}

func (c *ctxDB) QueryRow(ctx context.Context, sql string, arguments ...any) pgx.Row {
	if tx, ok := ExtractTx(ctx); ok {
		return tx.QueryRow(ctx, sql, arguments...)
	}
	return c.pool.QueryRow(ctx, sql, arguments...)
}

func (c *ctxDB) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	if tx, ok := ExtractTx(ctx); ok {
		return tx.CopyFrom(ctx, tableName, columnNames, rowSrc)
	}
	return c.pool.CopyFrom(ctx, tableName, columnNames, rowSrc)
}