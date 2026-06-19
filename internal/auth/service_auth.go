package auth

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/crypto"
	"bonfire-api/internal/repository"
	"bonfire-api/internal/user"
	"bonfire-api/internal/user_profile"
	"bonfire-api/internal/worker"
	"context"
	"encoding/json"
	"net/netip"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
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
	hashedPasswordBytes, err := crypto.HashPassword(req.Password)
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
	err = crypto.VerifyPassword(userAuth.PasswordHash, req.Password)
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

	// Generate session ID
	sessionID, err := uuid.NewV7()
	if err != nil {
		return LoginResponse{}, err
	}

	// Generate Refresh Token (7 days)
	refreshToken, err := s.generateRefreshToken(userID, sessionID)
	if err != nil {
		return LoginResponse{}, err
	}

	// Create user session
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

	// Return the tokens
	return LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// RotateTokens
func (s *AuthService) RotateTokens(ctx context.Context, oldTokenString string) (RotateTokensResponse, error) {
	// Check old token
	claims, err := s.tokenManager.VerifyJWT(oldTokenString, s.tokenConfig.RefreshSecret)
	if err != nil {
		return RotateTokensResponse{}, apperr.New(apperr.CodeUnauthenticated, "Invalid or expired session.")
	}

	// Check session
	sessionIDStr := claims.GetString("sid")
	if sessionIDStr == "" {
		return RotateTokensResponse{}, apperr.New(apperr.CodeUnauthenticated, "Malformed session payload.")
	}

	// Parse session id
	sessionUUID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		return RotateTokensResponse{}, apperr.New(apperr.CodeUnauthenticated, "Invalid session identifier.")
	}

	// Get user session
	session, err := s.store.UserSessionGetByID(ctx, pgtype.UUID{Bytes: sessionUUID, Valid: true})
	if err != nil {
		if repository.IsNotFoundError(err) {
			return RotateTokensResponse{}, apperr.New(apperr.CodeUnauthenticated, "Session no longer exists.")
		}
		return RotateTokensResponse{}, apperr.NewDBError(err)
	}

	// Check for an un-rotated token
	if session.RefreshToken != oldTokenString {
		_ = s.store.UserSessionMarkBlocked(ctx, session.ID)
		return RotateTokensResponse{}, apperr.New(apperr.CodeUnauthenticated, "Security breach detected. Session revoked.")
	}

	// Check if session blocked
	if session.IsBlocked {
		return RotateTokensResponse{}, apperr.New(apperr.CodeUnauthenticated, "Access denied. Session is blocked.")
	}

	// Check if session expired
	if time.Now().After(session.ExpiresAt.Time) {
		return RotateTokensResponse{}, apperr.New(apperr.CodeUnauthenticated, "Session expired. Please log in again.")
	}

	// Format userID
	userID := uuid.UUID(session.UserID.Bytes)

	// Get userRow
	userRow, err := s.store.UserGetByID(ctx, session.UserID)
	if err != nil {
		return RotateTokensResponse{}, apperr.NewDBError(err)
	}

	userIsVerified := userRow.VerifiedAt.Valid
	userRole := string(userRow.Role)

	// Generate new access token
	newAccessToken, err := s.generateAccessToken(userID, userRole, userIsVerified)
	if err != nil {
		return RotateTokensResponse{}, err
	}

	// Generate new refresh token
	newRefreshToken, err := s.generateRefreshToken(userID, sessionUUID)
	if err != nil {
		return RotateTokensResponse{}, err
	}

	// Update refresh token
	err = s.store.UserSessionUpdateRefreshToken(ctx, repository.UserSessionUpdateRefreshTokenParams{
		ID:           session.ID,
		RefreshToken: newRefreshToken,
		ExpiresAt: pgtype.Timestamptz{
			Time:  time.Now().Add(7 * 24 * time.Hour),
			Valid: true,
		},
	})
	if err != nil {
		return RotateTokensResponse{}, apperr.NewDBError(err)
	}

	// Return new tokens
	return RotateTokensResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
	}, nil
}
