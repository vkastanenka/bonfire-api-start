package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/crypto/bcrypt"

	"bonfire-api/internal/apperr"
	"bonfire-api/internal/auth"
	"bonfire-api/internal/repository"
)

func NewTestTokenConfig() auth.TokenConfig {
	return auth.TokenConfig{
		AccessSecret:        "access-secret",
		RefreshSecret:       "refresh-secret",
		VerificationSecret:  "verify-secret",
		PasswordResetSecret: "reset-secret",
		PasswordMFASecret:   "mfa-secret",
	}
}

func TestAuthService_Register(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := repository.NewMockStore(ctrl)

	// Assuming you have a constructor like NewAuthService
	authService := auth.NewAuthService(mockStore, NewTestTokenConfig())

	ctx := context.Background()
	displayName := "TestUser"
	req := auth.RegisterRequest{
		Email:       "test@example.com",
		Username:    "testuser",
		Password:    "supersecret123",
		DisplayName: &displayName,
	}

	t.Run("Success - User Registered Successfully", func(t *testing.T) {
		// 1. Mock availability check (both available)
		mockStore.EXPECT().
			UserCheckAvailability(ctx, repository.UserCheckAvailabilityParams{
				Email:    req.Email,
				Username: req.Username,
			}).
			Return(repository.UserCheckAvailabilityRow{
				EmailAvailable:    true,
				UsernameAvailable: true,
			}, nil)

		// 2. Mock the transaction execution
		mockStore.EXPECT().
			ExecTx(ctx, gomock.Any()).
			Return(nil)

		// Execute
		err := authService.Register(ctx, req)

		// Assert
		require.NoError(t, err)
	})

	t.Run("Conflict - Email and Username Taken", func(t *testing.T) {
		// Mock availability check (both taken)
		mockStore.EXPECT().
			UserCheckAvailability(ctx, repository.UserCheckAvailabilityParams{
				Email:    req.Email,
				Username: req.Username,
			}).
			Return(repository.UserCheckAvailabilityRow{
				EmailAvailable:    false,
				UsernameAvailable: false,
			}, nil)

		// ExecTx should NEVER be called in this scenario
		mockStore.EXPECT().ExecTx(gomock.Any(), gomock.Any()).Times(0)

		// Execute
		err := authService.Register(ctx, req)

		// Assert
		require.Error(t, err)
		var appErr *apperr.Error // Adjust to your specific error type implementation
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, apperr.CodeConflict, appErr.Code)
		assert.Contains(t, appErr.Details["fields"], "email")
		assert.Contains(t, appErr.Details["fields"], "username")
	})

	t.Run("Internal Error - DB Fails on Availability Check", func(t *testing.T) {
		mockErr := errors.New("db connection lost")

		mockStore.EXPECT().
			UserCheckAvailability(ctx, gomock.Any()).
			Return(repository.UserCheckAvailabilityRow{}, mockErr)

		// Execute
		err := authService.Register(ctx, req)

		// Assert
		require.Error(t, err)
		var appErr *apperr.Error
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, apperr.CodeInternal, appErr.Code)
	})

	t.Run("Internal Error - Transaction Fails", func(t *testing.T) {
		mockStore.EXPECT().
			UserCheckAvailability(ctx, gomock.Any()).
			Return(repository.UserCheckAvailabilityRow{
				EmailAvailable:    true,
				UsernameAvailable: true,
			}, nil)

		txErr := errors.New("transaction failed")
		mockStore.EXPECT().
			ExecTx(ctx, gomock.Any()).
			Return(txErr)

		// Execute
		err := authService.Register(ctx, req)

		// Assert
		require.Error(t, err)
		var appErr *apperr.Error
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, apperr.CodeInternal, appErr.Code)
	})
}

func TestAuthService_Login(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := repository.NewMockStore(ctrl)
	authService := auth.NewAuthService(mockStore, NewTestTokenConfig())

	ctx := context.Background()
	req := auth.LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}
	userAgent := "Mozilla/5.0"
	clientIP := "192.168.1.1"

	// Pre-hash password for the mock
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)

	t.Run("Success - Login Successful", func(t *testing.T) {
		// Mock User Retrieval
		mockStore.EXPECT().
			UserGetAuthCredentials(ctx, req.Email).
			Return(repository.UserGetAuthCredentialsRow{
				ID:           pgtype.UUID{Bytes: [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}, Valid: true},
				PasswordHash: string(hashedPassword),
			}, nil)

		// Mock Session Creation
		mockStore.EXPECT().
			UserSessionCreate(ctx, gomock.Any()).
			Return(repository.UserSession{}, nil)

		tokens, err := authService.Login(ctx, req, userAgent, clientIP)

		require.NoError(t, err)
		assert.NotEmpty(t, tokens["access_token"])
		assert.NotEmpty(t, tokens["refresh_token"])
	})

	t.Run("Failure - Invalid Password", func(t *testing.T) {
		mockStore.EXPECT().
			UserGetAuthCredentials(ctx, req.Email).
			Return(repository.UserGetAuthCredentialsRow{
				PasswordHash: string(hashedPassword),
			}, nil)

		// Password is wrong
		reqWrong := auth.LoginRequest{Email: req.Email, Password: "wrongpassword"}

		_, err := authService.Login(ctx, reqWrong, userAgent, clientIP)

		require.Error(t, err)
		// Check for unauthenticated error code
		var appErr *apperr.Error
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, apperr.CodeUnauthenticated, appErr.Code)
	})

	t.Run("Failure - User Not Found", func(t *testing.T) {
		mockStore.EXPECT().
			UserGetAuthCredentials(ctx, req.Email).
			Return(repository.UserGetAuthCredentialsRow{}, pgx.ErrNoRows)

		_, err := authService.Login(ctx, req, userAgent, clientIP)

		require.Error(t, err)
		var appErr *apperr.Error
		require.True(t, errors.As(err, &appErr))
		assert.Equal(t, apperr.CodeUnauthenticated, appErr.Code)
	})
}

func TestAuthService_RefreshAccessToken(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := repository.NewMockStore(ctrl)
	config := NewTestTokenConfig()
	authService := auth.NewAuthService(mockStore, config)

	ctx := context.Background()
	userID := [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}

	// NOTE: In a real test, use your token package to generate a signed string
	// that matches config.RefreshSecret. If you use a random string, VerifyJWT will fail.
	validToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." // Replace with a real signed JWT

	t.Run("Success - Token Rotated", func(t *testing.T) {
		mockStore.EXPECT().
			UserSessionGet(ctx, validToken).
			Return(repository.UserSession{
				ID:        pgtype.UUID{Bytes: userID, Valid: true},
				UserID:    pgtype.UUID{Bytes: userID, Valid: true},
				IsBlocked: false,
				ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(1 * time.Hour), Valid: true},
			}, nil)

		mockStore.EXPECT().
			UserSessionUpdateRefreshToken(ctx, gomock.Any()).
			Return(nil)

		tokens, err := authService.RefreshAccessToken(ctx, validToken)

		require.NoError(t, err)
		assert.NotEmpty(t, tokens["access_token"])
		assert.NotEmpty(t, tokens["refresh_token"])
	})

	t.Run("Failure - Session Blocked", func(t *testing.T) {
		mockStore.EXPECT().
			UserSessionGet(ctx, validToken).
			Return(repository.UserSession{
				IsBlocked: true,
			}, nil)

		_, err := authService.RefreshAccessToken(ctx, validToken)

		require.Error(t, err)
		var appErr *apperr.Error
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, apperr.CodeUnauthenticated, appErr.Code)
	})

	t.Run("Failure - Session Expired", func(t *testing.T) {
		mockStore.EXPECT().
			UserSessionGet(ctx, validToken).
			Return(repository.UserSession{
				ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(-1 * time.Hour), Valid: true},
			}, nil)

		_, err := authService.RefreshAccessToken(ctx, validToken)

		// 1. Verify that an error occurred
		require.Error(t, err)

		// 2. Perform the errors.As check
		var appErr *apperr.Error
		isAppErr := errors.As(err, &appErr)

		// 3. Assert that it IS the correct type, THEN inspect the fields
		require.True(t, isAppErr, "expected error to be of type *apperr.Error")
		assert.Equal(t, apperr.CodeUnauthenticated, appErr.Code)
	})
}
