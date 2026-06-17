package testing_helpers

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func NewPostgresContainer(t *testing.T) *pgxpool.Pool {
	ctx := context.Background()

	// 1. Start Postgres container
	pgContainer, err := postgres.Run(ctx,
		"postgres:18-alpine",
		postgres.WithDatabase("bonfire_test"),
		postgres.WithUsername("user"),
		postgres.WithPassword("password"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2),
		),
	)
	if err != nil {
		t.Fatal(err)
	}

	// 2. Get connection string
	connStr, _ := pgContainer.ConnectionString(ctx, "sslmode=disable")
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatal(err)
	}

	// 3. Run your migrations here
	// runMigrations(db)

	t.Cleanup(func() {
		pool.Close() // Close the pool, not just the connection
		pgContainer.Terminate(ctx)
	})

	return pool
}
