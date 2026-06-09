package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"

	"bonfire-api/internal/domain"
	"bonfire-api/internal/repository"
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

func (s *AuthService) Register(ctx context.Context, req RegisterRequest) (pgtype.UUID, error) {
	// 1. CPU heavy work out-of-band
	hashedPasswordBytes, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return pgtype.UUID{}, fmt.Errorf("hashing password failed: %w", err)
	}
	passwordHash := string(hashedPasswordBytes)

	var userID pgtype.UUID

	// 2. Wrap operations using our pristine, SQL-agnostic domain interface
	err = s.repo.WithTx(ctx, func(txCtx context.Context) error {
		// Extract the scoped transaction queries out of the context.
		// If WithTx wasn't called, this falls back to nil (or a base repo if preferred)
		q := repository.FromContext(txCtx, nil)
		if q == nil {
			return fmt.Errorf("database queries missing from transaction context")
		}

		userRow, err := q.CreateUser(txCtx, repository.CreateUserParams{
			Email:        req.Email,
			Username:     req.Username,
			PasswordHash: passwordHash,
		})
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return err // Or a custom domain error like domain.ErrRequestCanceled
			}
			if repository.IsUniqueViolation(err) {
				return ErrCredentialsUnavailable
			}
			return err
		}

		userID = userRow.ID

		if req.DisplayName != nil {
			_, err := q.CreateUserProfile(txCtx, repository.CreateUserProfileParams{
				UserID:      userRow.ID,
				DisplayName: pgtype.Text{String: *req.DisplayName, Valid: true},
			})
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return pgtype.UUID{}, err
	}

	// 3. Async downstream side-effects
	s.msgBroker.PublishUserRegisteredEvent(userID, req.Email)

	return userID, nil
}
