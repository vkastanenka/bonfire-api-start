package auth

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/repository"
	"bonfire-api/internal/token"
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"
)

type Store interface {
	ValidateUserCredentialsAvailability(ctx context.Context, arg repository.ValidateUserCredentialsAvailabilityParams) (repository.ValidateUserCredentialsAvailabilityRow, error)
	CreateOutboxEvent(ctx context.Context, arg repository.CreateOutboxEventParams) (repository.CreateOutboxEventRow, error)
	CreateSession(ctx context.Context, arg repository.CreateSessionParams) (repository.CreateSessionRow, error)
	CreateUser(ctx context.Context, arg repository.CreateUserParams) (repository.CreateUserRow, error)
	CreateUserProfile(ctx context.Context, arg repository.CreateUserProfileParams) (repository.CreateUserProfileRow, error)
	GetSession(ctx context.Context, arg string) (repository.GetSessionRow, error)
	GetUserByEmail(ctx context.Context, email string) (repository.GetUserByEmailRow, error)
	GetUserAuthCredentials(ctx context.Context, email string) (repository.GetUserAuthCredentialsRow, error)
	ExecTx(ctx context.Context, fn func(*repository.Queries) error) error
}

type TokenConfig struct {
	AccessSecret  string
	RefreshSecret string
}

type AuthService struct {
	store       Store
	tokenConfig TokenConfig
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

// Login
func (s *AuthService) Login(ctx context.Context, data LoginData, userAgent, clientIP string) (map[string]string, error) {
	unauthorizedErrDetails := map[string]string{
		"email":    "Invalid credentials.",
		"password": "Invalid credentials.",
	}

	// 1. Fetch user credentials
	userAuth, err := s.store.GetUserAuthCredentials(ctx, data.Email)
	if err != nil {
		// User not found
		if errors.Is(err, pgx.ErrNoRows) { // TODO: Abstract sql implementation details
			return nil, apperr.NewUnauthenticated("Invalid credentials.", apperr.WithDetails(unauthorizedErrDetails))
		}

		// Internal server error
		return nil, apperr.NewInternal(
			"An unexpected error occurred while verifying your account details.",
			apperr.WithErr(err),
		)
	}

	// 2. Compare the provided password with the stored hash
	err = bcrypt.CompareHashAndPassword([]byte(userAuth.PasswordHash), []byte(data.Password))
	if err != nil {
		return nil, apperr.NewUnauthenticated("Invalid credentials.", apperr.WithDetails(unauthorizedErrDetails))
	}

	// Convert pgtype.UUID to uuid.UUID by passing the underlying 16-byte array
	userID := uuid.UUID(userAuth.ID.Bytes)

	// 3. Generate Access Token (15 minutes)
	accessDuration := 15 * time.Minute
	accessToken, err := token.GenerateJWT(userID, s.tokenConfig.AccessSecret, accessDuration)
	if err != nil {
		return nil, apperr.NewInternal("Failed to generate access token.", apperr.WithErr(err))
	}

	// 4. Generate Refresh Token (7 days)
	refreshDuration := 7 * 24 * time.Hour
	refreshToken, err := token.GenerateJWT(userID, s.tokenConfig.RefreshSecret, refreshDuration)
	if err != nil {
		return nil, apperr.NewInternal("Failed to generate refresh token.", apperr.WithErr(err))
	}

	// 5. Store the session in the database
	_, err = s.store.CreateSession(ctx, repository.CreateSessionParams{
		UserID:       userAuth.ID,
		RefreshToken: refreshToken,
		UserAgent:    userAgent,
		ClientIp:     clientIP,
		IsBlocked:    false,
		ExpiresAt: pgtype.Timestamptz{
			Time:  time.Now().Add(refreshDuration),
			Valid: true,
		},
	})
	if err != nil {
		return nil, apperr.NewInternal("Failed to create user session.", apperr.WithErr(err))
	}

	// 6. Return the tokens
	return map[string]string{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	}, nil
}

// RefreshAccessToken validates the refresh token and issues a new access token.
func (s *AuthService) RefreshAccessToken(ctx context.Context, refreshToken string) (string, error) {
	// 1. Cryptographically verify the refresh token
	claims, err := token.VerifyJWT(refreshToken, s.tokenConfig.RefreshSecret)
	if err != nil {
		return "", apperr.NewUnauthenticated("Invalid or expired refresh token.")
	}

	// 2. Look up the session in the database
	session, err := s.store.GetSession(ctx, refreshToken)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", apperr.NewUnauthenticated("Session not found.")
		}
		return "", apperr.NewInternal("An unexpected error occurred while validating your session.", apperr.WithErr(err))
	}

	// 3. Validate the session state
	if session.IsBlocked {
		return "", apperr.NewUnauthenticated("Your session has been blocked.")
	}

	// Security check: Ensure the database user ID matches the token's payload user ID
	if claims.UserID != uuid.UUID(session.UserID.Bytes) {
		return "", apperr.NewUnauthenticated("Session identity mismatch.")
	}

	// Check database-level expiration as a fallback (JWT expiry is also checked in VerifyJWT)
	if time.Now().After(session.ExpiresAt.Time) {
		return "", apperr.NewUnauthenticated("Session expired. Please log in again.")
	}

	// 4. Issue a fresh Access Token (15 minutes)
	accessDuration := 15 * time.Minute
	userID := uuid.UUID(session.UserID.Bytes)

	newAccessToken, err := token.GenerateJWT(userID, s.tokenConfig.AccessSecret, accessDuration)
	if err != nil {
		return "", apperr.NewInternal("Failed to generate new access token.", apperr.WithErr(err))
	}

	return newAccessToken, nil
}
