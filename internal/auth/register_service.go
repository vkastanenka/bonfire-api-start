package auth

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/crypto"
	"bonfire-api/internal/repository"
	"bonfire-api/internal/user"
	"bonfire-api/internal/userprofile"
	"bonfire-api/internal/worker"
	"context"
)

// RegisterService
func (s *AuthService) Register(ctx context.Context, r RegisterParams) (RegisterResult, error) {
	// Define result
	var result RegisterResult

	// Check if credentials are available
	availability, err := s.store.UserCheckAvailability(ctx, repository.UserCheckAvailabilityParams{
		Email:    r.Email,
		Username: r.Username,
	})
	if err != nil {
		return RegisterResult{}, apperr.NewDBError(err)
	}

	// Cleanly handle conflict
	if !availability.EmailAvailable || !availability.UsernameAvailable {
		return RegisterResult{}, NewRegisterConflictError(availability.EmailAvailable, availability.UsernameAvailable)
	}

	// Hash password
	hashedPasswordBytes, err := crypto.HashPassword(r.Password)
	if err != nil {
		return RegisterResult{}, NewHashPasswordError(err)
	}
	passwordHash := string(hashedPasswordBytes)

	// Execute DB tx
	txErr := s.store.ExecTx(ctx, func(qtx *repository.Queries) error {
		// Create user
		userRow, err := qtx.UserCreate(ctx, repository.UserCreateParams{
			Email:        r.Email,
			Username:     r.Username,
			PasswordHash: passwordHash,
		})
		if err != nil {
			return err
		}

		// Set display name
		displayName := r.Username
		if r.DisplayName != nil && *r.DisplayName != "" {
			displayName = *r.DisplayName
		}

		// Create user profile
		userProfileRow, err := qtx.UserProfileCreate(ctx, repository.UserProfileCreateParams{
			UserID:      userRow.ID,
			DisplayName: displayName,
		})
		if err != nil {
			return err
		}

		// Create register event
		err = worker.EmitEvent(ctx, qtx, worker.EventUserRegistered, worker.AuthRegisterEventPayload{
			UserID: userRow.ID,
		})
		if err != nil {
			return err
		}

		result = RegisterResult{
			User:        user.NewView(userRow),
			UserProfile: userprofile.NewUserProfileView(userProfileRow),
		}

		return nil
	})

	// Handle tx errors
	if txErr != nil {
		return RegisterResult{}, apperr.NewDBError(txErr)
	}

	// Return result
	return result, nil
}
