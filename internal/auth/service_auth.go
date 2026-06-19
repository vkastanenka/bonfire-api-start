package auth

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/repository"
	"bonfire-api/internal/user"
	"bonfire-api/internal/user_profile"
	"bonfire-api/internal/worker"
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

// RegisterInput
type RegisterInput struct {
	Email       string
	Username    string
	DisplayName *string
	Password    string
}

// RegisterService
func (s *AuthService) Register(ctx context.Context, req RegisterInput) (user.UserResponse, user_profile.UserProfileResponse, error) {
	// Define user DTO
	var userResponse user.UserResponse
	var userProfileResponse user_profile.UserProfileResponse

	// Check if credentials are available
	availability, err := s.store.UserCheckAvailability(ctx, repository.UserCheckAvailabilityParams{
		Email:    req.Email,
		Username: req.Username,
	})
	if err != nil {
		return user.UserResponse{}, user_profile.UserProfileResponse{}, apperr.NewDBError(err)
	}

	// Append errors if credential conflict
	var availabilityErrors []apperr.ErrorOption
	if !availability.EmailAvailable {
		availabilityErrors = append(availabilityErrors, apperr.WithInvalidParam("email", ErrEmailTaken))
	}
	if !availability.UsernameAvailable {
		availabilityErrors = append(availabilityErrors, apperr.WithInvalidParam("username", ErrUsernameTaken))
	}

	// If credential conflicts, respond with error
	if len(availabilityErrors) > 0 {
		return user.UserResponse{}, user_profile.UserProfileResponse{}, apperr.New(
			apperr.CodeConflict,
			ErrRegFailed,
			availabilityErrors...,
		)
	}

	// Hash password
	hashedPasswordBytes, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return user.UserResponse{}, user_profile.UserProfileResponse{}, apperr.New(apperr.CodeInternal,
			ErrPasswordHashing,
			apperr.WithErr(err),
		)
	}
	passwordHash := string(hashedPasswordBytes)

	// Execute DB tx
	txErr := s.store.ExecTx(ctx, func(qtx *repository.Queries) error {
		// Create user
		userRow, err := qtx.UserCreate(ctx, repository.UserCreateParams{
			Email:        req.Email,
			Username:     req.Username,
			PasswordHash: passwordHash,
		})
		if err != nil {
			return err
		}

		// Set display name
		displayName := req.Username
		if req.DisplayName != nil && *req.DisplayName != "" {
			displayName = *req.DisplayName
		}

		// Create user profile
		userProfileRow, err := qtx.UserProfileCreate(ctx, repository.UserProfileCreateParams{
			UserID:      userRow.ID,
			DisplayName: displayName,
		})
		if err != nil {
			return err
		}

		// Format event payload
		eventPayload := worker.AuthRegisterEventPayload{
			UserID: userRow.ID,
		}

		jsonBytes, err := json.Marshal(eventPayload)
		if err != nil {
			return err
		}

		// Create event
		_, err = qtx.OutboxEventCreate(ctx, repository.OutboxEventCreateParams{
			EventType: EventUserRegistered,
			Payload:   jsonBytes,
		})
		if err != nil {
			return err
		}

		// Set response DTOs
		userResponse = user.CreateUserResponse(userRow)
		userProfileResponse = user_profile.CreateUserProfileResponse(userProfileRow)

		return nil
	})

	// Handle tx errors
	if txErr != nil {
		return user.UserResponse{}, user_profile.UserProfileResponse{}, apperr.NewDBError(txErr)
	}

	// Return created resources
	return userResponse, userProfileResponse, nil
}

// LoginInput
type LoginInput struct {
	Email    string
	Password string
}

// Login
func (s *AuthService) Login(ctx context.Context, req LoginInput, userAgent string, clientIP netip.Addr) (LoginResponse, error) {
	// Set up invalid params for error handling
	invalidParams := apperr.WithInvalidParams([]apperr.InvalidParam{
		{Name: "email", Reason: "Invalid credentials."},
		{Name: "password", Reason: "Invalid credentials."},
	})

	// Fetch user credentials
	userAuth, err := s.store.UserGetAuthCredentials(ctx, req.Email)
	if err != nil {
		// User not found
		if repository.IsNotFoundError(err) {
			return LoginResponse{}, apperr.New(apperr.CodeNotFound, "Invalid credentials.", invalidParams)
		}

		return LoginResponse{}, apperr.NewDBError(err)
	}

	// Check password
	err = bcrypt.CompareHashAndPassword([]byte(userAuth.PasswordHash), []byte(req.Password))
	if err != nil {
		return LoginResponse{}, apperr.New(apperr.CodeUnauthenticated, "Invalid credentials.", invalidParams)
	}

	// Convert pgtype.UUID to uuid.UUID
	userID := uuid.UUID(userAuth.ID.Bytes)
	userIsVerified := userAuth.VerifiedAt.Valid
	userRole := string(userAuth.Role)

	// Generate Access Token (15 minutes)
	accessToken, err := s.generateAccessToken(userID, userRole, userIsVerified)
	if err != nil {
		return LoginResponse{}, err
	}

	// Pre-generate the Session ID (UUIDv7) to break the dependency cycle
	sessionID, err := uuid.NewV7()
	if err != nil {
		return LoginResponse{}, err
	}

	// 4. Generate Refresh Token (7 days) embedding the pre-generated sessionID
	refreshToken, err := s.generateRefreshToken(userID, sessionID)
	if err != nil {
		return LoginResponse{}, err
	}

	// 5. Store the session in the database using the explicit sessionID
	_, err = s.store.UserSessionCreate(ctx, repository.UserSessionCreateParams{
		ID:           pgtype.UUID{Bytes: sessionID, Valid: true},
		UserID:       userAuth.ID,
		RefreshToken: refreshToken,
		UserAgent:    userAgent,
		ClientIp:     clientIP,
		IsBlocked:    false,
		ExpiresAt: pgtype.Timestamptz{
			Time:  time.Now().Add(7 * 24 * time.Hour),
			Valid: true,
		},
	})
	if err != nil {
		return LoginResponse{}, apperr.NewDBError(err)
	}

	// 6. Return the tokens
	return LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
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
