package auth

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/repository"
	"context"

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
	// Step 1. Execute fast-path availability pre-check
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

	// Gather explicit field violations
	details := make(map[string]string)
	if !availability.EmailAvailable {
		details["email"] = "This email address is already registered."
	}
	if !availability.UsernameAvailable {
		details["username"] = "This username is already taken."
	}

	// If there are any violations, return a conflict error with structured details
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

	// Suppress compiler warning for the next phase
	_ = passwordHash

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
