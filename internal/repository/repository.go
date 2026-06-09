package repository

import (
	"context"
)

// Define a private, unexported custom type for the context key.
// This prevents key collisions with any other package using the same context.
type ctxKey string

const queriesContextKey ctxKey = "sqlc_queries"

// FromContext extracts the *Queries instance from the context.
// If no transaction or query instance is found, it falls back to the provided fallback.
func FromContext(ctx context.Context, fallback *Queries) *Queries {
	if q, ok := ctx.Value(queriesContextKey).(*Queries); ok {
		return q
	}
	return fallback
}

// InjectQueries creates a new child context containing the *Queries instance.
func InjectQueries(ctx context.Context, q *Queries) context.Context {
	return context.WithValue(ctx, queriesContextKey, q)
}
