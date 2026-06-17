//go:build integration

package auth_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"bonfire-api/internal/auth"
	"bonfire-api/internal/repository"
	"bonfire-api/internal/testing_helpers"
	"bonfire-api/internal/token"
)

func TestAuthService_Register_Integration(t *testing.T) {
	// 1. Setup real infrastructure
	db := testing_helpers.NewPostgresContainer(t)
	store := repository.NewStore(db)

	// Use a real token manager implementation if available,
	// or a functional one that doesn't rely on complex external dependencies.
	tokenManager := token.NewJWTManager()
	config := auth.TokenConfig{
		AccessSecret:  "test-secret",
		RefreshSecret: "refresh-secret",
	}

	authService := auth.NewAuthService(store, tokenManager, config)

	t.Run("Success - User persisted in database", func(t *testing.T) {
		ctx := context.Background()
		displayName := "Integration Test User"
		req := auth.RegisterRequest{
			Email:       "integration@example.com",
			Username:    "int_user_01",
			Password:    "securePassword123",
			DisplayName: &displayName,
		}

		// Act: Run real service layer
		err := authService.Register(ctx, req)
		require.NoError(t, err)

		// Assert: Verify database state directly
		// This tests your SQL queries, constraints, and transaction logic
		var userID string
		var storedEmail string
		err = db.QueryRow(ctx,
			"SELECT id, email FROM users WHERE username = $1",
			req.Username).Scan(&userID, &storedEmail)

		require.NoError(t, err, "User should exist in the database")
		assert.Equal(t, req.Email, storedEmail)

		// Verify password hash is actually stored (not plain text)
		var storedHash string
		err = db.QueryRow(ctx, "SELECT password_hash FROM users WHERE id = $1", userID).Scan(&storedHash)
		require.NoError(t, err)
		assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(req.Password)), "Stored hash should match password")
	})

	t.Run("Failure - Duplicate Username Fails", func(t *testing.T) {
		ctx := context.Background()
		req := auth.RegisterRequest{
			Email:    "new@example.com",
			Username: "int_user_01", // Already exists from previous test
			Password: "password123",
		}

		err := authService.Register(ctx, req)
		require.Error(t, err, "Registration should fail for duplicate username")
	})
}
