package auth

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/repository"
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"
)

type Store interface {
	ValidateUserCredentialsAvailability(ctx context.Context, arg repository.ValidateUserCredentialsAvailabilityParams) (repository.ValidateUserCredentialsAvailabilityRow, error)
	CreateUser(ctx context.Context, arg repository.CreateUserParams) (repository.CreateUserRow, error)
	CreateUserProfile(ctx context.Context, arg repository.CreateUserProfileParams) (repository.CreateUserProfileRow, error)
	CreateOutboxEvent(ctx context.Context, arg repository.CreateOutboxEventParams) (repository.CreateOutboxEventRow, error)
	ExecTx(ctx context.Context, fn func(*repository.Queries) error) error
}

type AuthService struct {
	store Store
}

func NewAuthService(store Store) *AuthService {
	return &AuthService{store: store}
}

// Register runs the business logic for creating a new user account.
func (s *AuthService) Register(ctx context.Context, data RegisterData) error {
	// Step 1. Credentials availability validation

	// 1a. Execute fast-path availability pre-check
	availability, err := s.store.ValidateUserCredentialsAvailability(ctx, repository.ValidateUserCredentialsAvailabilityParams{
		Email:    data.Email,
		Username: data.Username,
	})
	if err != nil {
		return apperr.NewInternal(
			"An unexpected error occurred while verifying your account details.",
			apperr.WithErr(err),
		)
	}

	// 1b. Gather explicit field violations
	details := make(map[string]string)
	if !availability.EmailAvailable {
		details["email"] = "This email address is already registered."
	}
	if !availability.UsernameAvailable {
		details["username"] = "This username is already taken."
	}

	// 1c. If there are any violations, return a conflict error with structured details
	if len(details) > 0 {
		return apperr.NewConflict(
			"Registration failed due to unavailable credentials.",
			apperr.WithDetails(details),
		)
	}

	// Step 2: Password Hashing (Securely inside the Service layer!)
	hashedPasswordBytes, err := bcrypt.GenerateFromPassword([]byte(data.Password), bcrypt.DefaultCost)
	if err != nil {
		return apperr.NewInternal(
			"An unexpected error occurred while securing your account password.",
			apperr.WithErr(err),
		)
	}
	passwordHash := string(hashedPasswordBytes)

	// Step 3: Execute DB transaction (CreateUser + CreateUserProfile)
	// We pass the transaction block callback. Notice it uses the decoupled `qtx` (*repository.Queries) instance.
	txErr := s.store.ExecTx(ctx, func(qtx *repository.Queries) error {
		// 3a. Insert the core user record
		userRow, err := qtx.CreateUser(ctx, repository.CreateUserParams{
			Email:        data.Email,
			Username:     data.Username,
			PasswordHash: passwordHash,
		})
		if err != nil {
			return err // Bubbles back to ExecTx to trigger an automatic Rollback
		}

		// Determine a fallback display name if none was provided in the request payload
		displayName := data.Username
		if data.DisplayName != nil && *data.DisplayName != "" {
			displayName = *data.DisplayName
		}

		// 3b. Insert the accompanying profile record, linking it via the new user's ID
		_, err = qtx.CreateUserProfile(ctx, repository.CreateUserProfileParams{
			UserID:      userRow.ID,
			DisplayName: pgtype.Text{String: displayName, Valid: true},
		})
		if err != nil {
			return err // Bubbles back to ExecTx to trigger an automatic Rollback
		}

		// 3c. Marshal the specialized payload map into a dynamic JSON byte slice
		eventPayload := map[string]string{
			"email":    data.Email,
			"username": data.Username,
		}
		jsonBytes, err := json.Marshal(eventPayload)
		if err != nil {
			return err
		}

		// 3d. Append the operational notification intent directly inside the transaction log
		_, err = qtx.CreateOutboxEvent(ctx, repository.CreateOutboxEventParams{
			EventType: "user.registered",
			Payload:   jsonBytes,
		})
		if err != nil {
			return err
		}

		return nil // Everything succeeded, ExecTx will attempt a Commit
	})

	// Handle transactional completion states
	if txErr != nil {
		return apperr.NewInternal(
			"An unexpected error occurred while creating your account. Please try again.",
			apperr.WithErr(txErr),
		)
	}

	return nil
}
