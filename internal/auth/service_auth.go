package auth

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/repository"
	"context"
	"encoding/json"
	"errors"
	"net/netip"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"
)

// Register runs the business logic for creating a new user account.
func (s *AuthService) Register(ctx context.Context, req RegisterRequest) error {
	// Step 1. Credentials availability validation

	// 1a. Execute fast-path availability pre-check
	availability, err := s.store.UserCheckAvailability(ctx, repository.UserCheckAvailabilityParams{
		Email:    req.Email,
		Username: req.Username,
	})
	if err != nil {
		return apperr.New(apperr.CodeInternal,
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
		return apperr.New(apperr.CodeConflict,
			"Registration failed due to unavailable credentials.",
			apperr.WithDetails("fields", details),
		)
	}

	// Step 2: Password Hashing (Securely inside the Service layer!)
	hashedPasswordBytes, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return apperr.New(apperr.CodeInvalidInput,
			"An unexpected error occurred while securing your account password.",
			apperr.WithErr(err),
		)
	}
	passwordHash := string(hashedPasswordBytes)

	// Step 3: Execute DB transaction (CreateUser + CreateUserProfile)
	// We pass the transaction block callback. Notice it uses the decoupled `qtx` (*repository.Queries) instance.
	txErr := s.store.ExecTx(ctx, func(qtx *repository.Queries) error {
		// 3a. Insert the core user record
		userRow, err := qtx.UserCreate(ctx, repository.UserCreateParams{
			Email:        req.Email,
			Username:     req.Username,
			PasswordHash: passwordHash,
		})
		if err != nil {
			return err // Bubbles back to ExecTx to trigger an automatic Rollback
		}

		// Determine a fallback display name if none was provided in the request payload
		displayName := req.Username
		if req.DisplayName != nil && *req.DisplayName != "" {
			displayName = *req.DisplayName
		}

		// 3b. Insert the accompanying profile record, linking it via the new user's ID
		_, err = qtx.UserProfileCreate(ctx, repository.UserProfileCreateParams{
			UserID:      userRow.ID,
			DisplayName: pgtype.Text{String: displayName, Valid: true},
		})
		if err != nil {
			return err // Bubbles back to ExecTx to trigger an automatic Rollback
		}

		// 3c. Marshal the specialized payload map into a dynamic JSON byte slice
		eventPayload := map[string]string{
			"email":    req.Email,
			"username": req.Username,
		}
		jsonBytes, err := json.Marshal(eventPayload)
		if err != nil {
			return err
		}

		// 3d. Append the operational notification intent directly inside the transaction log
		_, err = qtx.OutboxEventCreate(ctx, repository.OutboxEventCreateParams{
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
		return apperr.New(apperr.CodeInternal,
			"An unexpected error occurred while creating your account. Please try again.",
			apperr.WithErr(txErr),
		)
	}

	return nil
}

// Login
func (s *AuthService) Login(ctx context.Context, req LoginRequest, userAgent, clientIP string) (map[string]string, error) {
	unauthorizedErrDetails := map[string]string{
		"email":    "Invalid credentials.",
		"password": "Invalid credentials.",
	}

	// 1. Fetch user credentials
	userAuth, err := s.store.UserGetAuthCredentials(ctx, req.Email)
	if err != nil {
		// User not found
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.New(apperr.CodeUnauthenticated, "Invalid credentials.", apperr.WithDetails("fields", unauthorizedErrDetails))
		}

		// Internal server error
		return nil, apperr.New(apperr.CodeInternal,
			"An unexpected error occurred while verifying your account details.",
			apperr.WithErr(err),
		)
	}

	// 2. Compare the provided password with the stored hash
	err = bcrypt.CompareHashAndPassword([]byte(userAuth.PasswordHash), []byte(req.Password))
	if err != nil {
		return nil, apperr.New(apperr.CodeUnauthenticated, "Invalid credentials.", apperr.WithDetails("fields", unauthorizedErrDetails))
	}

	// Convert pgtype.UUID to uuid.UUID by passing the underlying 16-byte array
	userID := uuid.UUID(userAuth.ID.Bytes)

	// 3. Generate Access Token (15 minutes)
	accessToken, err := s.generateAccessToken(userID)
	if err != nil {
		return nil, err
	}

	// 4. Generate Refresh Token (7 days)
	refreshToken, err := s.generateRefreshToken(userID)
	if err != nil {
		return nil, err
	}

	ipAddr, err := netip.ParseAddr(clientIP)
	if err != nil {
		return nil, apperr.New(apperr.CodeInvalidInput, "Invalid IP address format.", apperr.WithErr(err))
	}

	// 5. Store the session in the database
	_, err = s.store.UserSessionCreate(ctx, repository.UserSessionCreateParams{
		UserID:       userAuth.ID,
		RefreshToken: refreshToken,
		UserAgent:    userAgent,
		ClientIp:     ipAddr,
		IsBlocked:    false,
		ExpiresAt: pgtype.Timestamptz{
			Time:  time.Now().Add(7 * 24 * time.Hour),
			Valid: true,
		},
	})
	if err != nil {
		return nil, apperr.New(apperr.CodeInternal, "Failed to create user session.", apperr.WithErr(err))
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
	claims, err := s.tokenManager.VerifyJWT(oldRefreshToken, s.tokenConfig.RefreshSecret)
	if err != nil {
		return nil, apperr.New(apperr.CodeUnauthenticated, "Invalid or expired refresh token.")
	}

	// 2. Look up the session using the old token
	session, err := s.store.UserSessionGet(ctx, oldRefreshToken)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.New(apperr.CodeUnauthenticated, "Session not found or token already consumed.")
		}
		return nil, apperr.New(apperr.CodeInternal, "An unexpected error occurred while validating your session.", apperr.WithErr(err))
	}

	// 3. Validate the session state
	if session.IsBlocked {
		return nil, apperr.New(apperr.CodeUnauthenticated, "Your session has been blocked.")
	}

	if claims.UserID != uuid.UUID(session.UserID.Bytes) {
		return nil, apperr.New(apperr.CodeUnauthenticated, "Session identity mismatch.")
	}

	if time.Now().After(session.ExpiresAt.Time) {
		return nil, apperr.New(apperr.CodeUnauthenticated, "Session expired. Please log in again.")
	}

	// 4. Issue a fresh Access Token (15 minutes)
	accessDuration := 15 * time.Minute
	userID := uuid.UUID(session.UserID.Bytes)

	newAccessToken, err := s.tokenManager.GenerateJWT(userID, s.tokenConfig.AccessSecret, accessDuration)
	if err != nil {
		return nil, apperr.New(apperr.CodeInternal, "Failed to generate new access token.", apperr.WithErr(err))
	}

	// 5. Issue a fresh Refresh Token (Reset the 7-day clock)
	refreshDuration := 7 * 24 * time.Hour
	newRefreshToken, err := s.tokenManager.GenerateJWT(userID, s.tokenConfig.RefreshSecret, refreshDuration)
	if err != nil {
		return nil, apperr.New(apperr.CodeInternal, "Failed to generate new refresh token.", apperr.WithErr(err))
	}

	// 6. ROTATION: Update the database with the new refresh token
	err = s.store.UserSessionUpdateRefreshToken(ctx, repository.UserSessionUpdateRefreshTokenParams{
		ID:           session.ID,
		RefreshToken: newRefreshToken,
		ExpiresAt: pgtype.Timestamptz{
			Time:  time.Now().Add(refreshDuration),
			Valid: true,
		},
	})
	if err != nil {
		return nil, apperr.New(apperr.CodeInternal, "Failed to rotate session tokens.", apperr.WithErr(err))
	}

	return map[string]string{
		"access_token":  newAccessToken,
		"refresh_token": newRefreshToken,
	}, nil
}
