package auth_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"bonfire-api/internal/apperr"     // Adjust path to your apperr package
	"bonfire-api/internal/auth"       // Adjust path to your auth service package
	"bonfire-api/internal/repository" // Adjust path to your repository package
)

func TestAuthService_Register(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := repository.NewMockStore(ctrl)

	// Assuming you have a constructor like NewAuthService
	authService := auth.NewAuthService(mockStore)

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
