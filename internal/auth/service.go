package auth

import (
	"bonfire-api/internal/domain"
)

type AuthService struct {
	repo      domain.DBRepository
	msgBroker domain.MessageBroker
}

func NewAuthService(repo domain.DBRepository, broker domain.MessageBroker) *AuthService {
	return &AuthService{
		repo:      repo,
		msgBroker: broker,
	}
}

// func (s *AuthService) Register(ctx context.Context, req RegisterRequest) (pgtype.UUID, error) {
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
// 		if req.DisplayName != nil {
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
