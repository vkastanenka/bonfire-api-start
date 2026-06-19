package auth

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/crypto"
	"bonfire-api/internal/repository"
	"bonfire-api/internal/user"
	"bonfire-api/internal/user_profile"
	"bonfire-api/internal/worker"
	"context"
)

// RegisterService
func (s *AuthService) Register(ctx context.Context, r RegisterParams) (RegisterResult, error) {
	// Define user DTO
	var userResponse user.UserResponse
	var userProfileResponse user_profile.UserProfileResponse

	// Check if credentials are available
	availability, err := s.store.UserCheckAvailability(ctx, repository.UserCheckAvailabilityParams{
		Email:    r.Email,
		Username: r.Username,
	})
	if err != nil {
		return RegisterResult{
			User:        user.UserResponse{},
			UserProfile: user_profile.UserProfileResponse{},
		}, apperr.NewDBError(err)
	}

	// Append errors if credential conflict
	var availabilityErrors []apperr.ErrorOption
	if !availability.EmailAvailable {
		availabilityErrors = append(availabilityErrors, apperr.WithInvalidParam("email", ErrEmailTaken))
	}
	if !availability.UsernameAvailable {
		availabilityErrors = append(availabilityErrors, apperr.WithInvalidParam("username", ErrUsernameTaken))
	}

	// If credential conflicts, respond with error
	if len(availabilityErrors) > 0 {
		return RegisterResult{
				User:        user.UserResponse{},
				UserProfile: user_profile.UserProfileResponse{},
			}, apperr.New(
				apperr.CodeConflict,
				ErrRegFailed,
				availabilityErrors...,
			)
	}

	// Hash password
	hashedPasswordBytes, err := crypto.HashPassword(r.Password)
	if err != nil {
		return RegisterResult{
				User:        user.UserResponse{},
				UserProfile: user_profile.UserProfileResponse{},
			}, apperr.New(apperr.CodeInternal,
				ErrHashPassword,
				apperr.WithErr(err),
			)
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

		// Set response DTOs
		userResponse = user.CreateUserResponse(userRow)
		userProfileResponse = user_profile.CreateUserProfileResponse(userProfileRow)

		return nil
	})

	// Handle tx errors
	if txErr != nil {
		return RegisterResult{
			User:        user.UserResponse{},
			UserProfile: user_profile.UserProfileResponse{},
		}, apperr.NewDBError(txErr)
	}

	// Return created resources
	return RegisterResult{
		User:        userResponse,
		UserProfile: userProfileResponse,
	}, nil
}
