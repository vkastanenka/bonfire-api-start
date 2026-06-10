package auth

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/repository"
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"
)

type Store interface {
	ValidateUserCredentialsAvailability(ctx context.Context, arg repository.ValidateUserCredentialsAvailabilityParams) (repository.ValidateUserCredentialsAvailabilityRow, error)
	CreateUser(ctx context.Context, arg repository.CreateUserParams) (repository.CreateUserRow, error)
	CreateUserProfile(ctx context.Context, arg repository.CreateUserProfileParams) (repository.CreateUserProfileRow, error)

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

// func (s *AuthService) Register(ctx context.Context, data RegisterData) (pgtype.UUID, error) {
// 	// Hash password
// 	hashedPasswordBytes, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
// 	if err != nil {
// 		return pgtype.UUID{}, fmt.Errorf("hashing password failed: %w", err)
// 	}
// 	passwordHash := string(hashedPasswordBytes)

// 	var userID pgtype.UUID

// 	// Create transaction
// 	err = s.repo.WithTx(ctx, func(tx domain.TxUnit) error {

// 		// Directly call CreateUser on the transactional unit using primitive parameters
// 		var txErr error
// 		userID, txErr = tx.CreateUser(ctx, req.Email, req.Username, passwordHash)
// 		if txErr != nil {
// 			if errors.Is(txErr, context.Canceled) {
// 				return txErr
// 			}
// 			// Note: The infrastructure layer implementing tx.CreateUser should
// 			// map DB-specific unique violations to a generic domain error, or you can
// 			// inspect it here if your repo passes specialized errors back.
// 			return txErr
// 		}

// 		// Conditionally create the user profile if a DisplayName was provided
// 		if data.DisplayName != nil {
// 			_, txErr = tx.CreateUserProfile(ctx, userID, req.DisplayName)
// 			if txErr != nil {
// 				return txErr
// 			}
// 		}

// 		return nil
// 	})

// 	if err != nil {
// 		return pgtype.UUID{}, err
// 	}

// 	// 3. Async downstream side-effects
// 	s.msgBroker.PublishUserRegisteredEvent(userID, req.Email)

// 	return userID, nil
// }
