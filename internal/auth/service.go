package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"bonfire-api/internal/repository"
)

var (
	ErrInvalidCredentials = errors.New("Invalid credentials.")
)

type AuthService struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *AuthService {
	return &AuthService{pool: pool}
}

func (s *AuthService) Register(ctx context.Context, req RegisterRequest) error {
	queries := repository.New(s.pool)

	availability, err := queries.ValidateUserCredentialsAvailability(ctx, repository.ValidateUserCredentialsAvailabilityParams{
		Email:    req.Email,
		Username: req.Username,
	})
	if err != nil {
		return fmt.Errorf("checking credentials availability: %w", err)
	}

	if !availability.Email || !availability.Username {
		return ErrInvalidCredentials
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hashing password failed: %w", err)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("starting transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	txQueries := repository.New(tx)

	userRow, err := txQueries.CreateUser(ctx, repository.CreateUserParams{
		Email:        req.Email,
		Username:     req.Username,
		PasswordHash: string(hashedPassword),
	})

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrInvalidCredentials
		}
		return fmt.Errorf("inserting user row: %w", err)
	}

	if req.DisplayName != "" {
		_, err := txQueries.CreateUserProfile(ctx, repository.CreateUserProfileParams{
			UserID: userRow.ID,
			DisplayName: pgtype.Text{
				String: req.DisplayName,
				Valid:  true,
			},
		})
		if err != nil {
			return fmt.Errorf("inserting user profile row: %w", err)
		}
	}

	return tx.Commit(ctx)
}
