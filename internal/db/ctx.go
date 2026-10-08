package db

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// txKey is a unique key for storing active pgx.Tx instances within ctx.
type txKey struct{}

func InjectTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}

func ExtractTx(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(txKey{}).(pgx.Tx)
	return tx, ok
}

func IsTx(ctx context.Context) bool {
	_, ok := ExtractTx(ctx)
	return ok
}

// ctxDB wraps a pgxpool.Pool and routes operations to a transaction in ctx or the pool.
type ctxDB struct {
	pool *pgxpool.Pool
}

func (c *ctxDB) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	return c.runner(ctx).Exec(ctx, sql, arguments...)
}

func (c *ctxDB) Query(ctx context.Context, sql string, arguments ...any) (pgx.Rows, error) {
	return c.runner(ctx).Query(ctx, sql, arguments...)
}

func (c *ctxDB) QueryRow(ctx context.Context, sql string, arguments ...any) pgx.Row {
	return c.runner(ctx).QueryRow(ctx, sql, arguments...)
}

func (c *ctxDB) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	return c.runner(ctx).CopyFrom(ctx, tableName, columnNames, rowSrc)
}

func (c *ctxDB) runner(ctx context.Context) DBTX {
	if tx, ok := ExtractTx(ctx); ok {
		return tx
	}
	return c.pool
}
