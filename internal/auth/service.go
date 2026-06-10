package auth

import (
	"bonfire-api/internal/repository"
	"context"
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

func (s *AuthService) Register() {

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
