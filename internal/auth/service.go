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
	UpdateSessionRefreshToken(ctx context.Context, arg repository.UpdateSessionRefreshTokenParams) error
	UpdateUserPassword(ctx context.Context, arg repository.UpdateUserPasswordParams) error
	VerifyUserEmail(ctx context.Context, arg repository.VerifyUserEmailParams) error
	ExecTx(ctx context.Context, fn func(*repository.Queries) error) error
}

type TokenConfig struct {
	AccessSecret        string
	RefreshSecret       string
	VerificationSecret  string
	PasswordResetSecret string
}

type AuthService struct {
	store       Store
	tokenConfig TokenConfig
}

func NewAuthService(store Store, tokenConfig TokenConfig) *AuthService {
	return &AuthService{store: store, tokenConfig: tokenConfig}
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
			Flags:        int64(UserFlagNone),
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

func (s *AuthService) VerifyEmail(ctx context.Context, tokenStr string) error {
	// 1. Validate the stateless token structure
	claims, err := token.VerifyJWT(tokenStr, s.tokenConfig.VerificationSecret)
	if err != nil {
		return apperr.NewUnauthenticated("The verification link is invalid or has expired.")
	}

	var pgUserID pgtype.UUID
	pgUserID.Bytes = claims.UserID
	pgUserID.Valid = true

	// 2. Perform safe, atomic bitwise alteration
	err = s.store.VerifyUserEmail(ctx, repository.VerifyUserEmailParams{
		ID:    pgUserID,
		Flags: int64(UserFlagVerified), // Merges bit value 1 via bitwise OR
	})
	if err != nil {
		return apperr.NewInternal("Failed to update verification flags.", apperr.WithErr(err))
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

// RefreshAccessToken validates the refresh token, rotates it, and issues fresh tokens.
func (s *AuthService) RefreshAccessToken(ctx context.Context, oldRefreshToken string) (map[string]string, error) {
	// 1. Cryptographically verify the old refresh token
	claims, err := token.VerifyJWT(oldRefreshToken, s.tokenConfig.RefreshSecret)
	if err != nil {
		return nil, apperr.NewUnauthenticated("Invalid or expired refresh token.")
	}

	// 2. Look up the session using the old token
	session, err := s.store.GetSession(ctx, oldRefreshToken)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// REPLAY DETECTION: If the JWT is valid but not in the DB, it means
			// the token was already used (rotated) or the user logged out.
			return nil, apperr.NewUnauthenticated("Session not found or token already consumed.")
		}
		return nil, apperr.NewInternal("An unexpected error occurred while validating your session.", apperr.WithErr(err))
	}

	// 3. Validate the session state
	if session.IsBlocked {
		return nil, apperr.NewUnauthenticated("Your session has been blocked.")
	}

	if claims.UserID != uuid.UUID(session.UserID.Bytes) {
		return nil, apperr.NewUnauthenticated("Session identity mismatch.")
	}

	if time.Now().After(session.ExpiresAt.Time) {
		return nil, apperr.NewUnauthenticated("Session expired. Please log in again.")
	}

	// 4. Issue a fresh Access Token (15 minutes)
	accessDuration := 15 * time.Minute
	userID := uuid.UUID(session.UserID.Bytes)

	newAccessToken, err := token.GenerateJWT(userID, s.tokenConfig.AccessSecret, accessDuration)
	if err != nil {
		return nil, apperr.NewInternal("Failed to generate new access token.", apperr.WithErr(err))
	}

	// 5. Issue a fresh Refresh Token (Reset the 7-day clock)
	refreshDuration := 7 * 24 * time.Hour
	newRefreshToken, err := token.GenerateJWT(userID, s.tokenConfig.RefreshSecret, refreshDuration)
	if err != nil {
		return nil, apperr.NewInternal("Failed to generate new refresh token.", apperr.WithErr(err))
	}

	// 6. ROTATION: Update the database with the new refresh token
	err = s.store.UpdateSessionRefreshToken(ctx, repository.UpdateSessionRefreshTokenParams{
		ID:           session.ID,
		RefreshToken: newRefreshToken,
		ExpiresAt: pgtype.Timestamptz{
			Time:  time.Now().Add(refreshDuration),
			Valid: true,
		},
	})
	if err != nil {
		return nil, apperr.NewInternal("Failed to rotate session tokens.", apperr.WithErr(err))
	}

	return map[string]string{
		"access_token":  newAccessToken,
		"refresh_token": newRefreshToken,
	}, nil
}

func (s *AuthService) ForgotPassword(ctx context.Context, email string) error {
	user, err := s.store.GetUserByEmail(ctx, email)
	if err != nil {
		// If not found, return nil (don't leak user existence)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return apperr.NewInternal("System error", apperr.WithErr(err))
	}

	// Generate a short-lived token (15 mins) specifically for resetting
	userID := uuid.UUID(user.ID.Bytes)
	resetToken, err := token.GenerateJWT(userID, s.tokenConfig.PasswordResetSecret, 15*time.Minute)
	if err != nil {
		return err
	}

	// Create Outbox Event
	jsonBytes, _ := json.Marshal(map[string]string{"email": email, "token": resetToken})
	_, err = s.store.CreateOutboxEvent(ctx, repository.CreateOutboxEventParams{
		EventType: "user.forgot_password",
		Payload:   jsonBytes,
	})
	return err
}

func (s *AuthService) ResetPassword(ctx context.Context, tokenStr string, newPassword string) error {
	// Verify the token using the PasswordResetSecret
	claims, err := token.VerifyJWT(tokenStr, s.tokenConfig.PasswordResetSecret)
	if err != nil {
		return apperr.NewUnauthenticated("Invalid or expired reset token.")
	}

	// Hash the new password
	hashedPasswordBytes, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return apperr.NewInternal("Failed to hash password", apperr.WithErr(err))
	}

	// Execute update
	err = s.store.UpdateUserPassword(ctx, repository.UpdateUserPasswordParams{
		ID:           pgtype.UUID{Bytes: claims.UserID, Valid: true},
		PasswordHash: string(hashedPasswordBytes),
	})
	if err != nil {
		return apperr.NewInternal("Failed to update password", apperr.WithErr(err))
	}

	return nil
}

// TODO: ResendUserVerification

// TODO: 2FA

// TODO: DeviceVerification

// TODO: Handle phones

// TODO: Handle QR codes
