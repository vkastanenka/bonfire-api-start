package auth

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/repository"
	"bonfire-api/internal/token"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pquerna/otp/totp"
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
	EnableUserTOTP(ctx context.Context, arg repository.EnableUserTOTPParams) error
	DisableUserTOTP(ctx context.Context, id pgtype.UUID) error
	GetUserTOTPSecret(ctx context.Context, id pgtype.UUID) (pgtype.Text, error)         // ADD THIS
	GetUserByID(ctx context.Context, id pgtype.UUID) (repository.GetUserByIDRow, error) // ADD THIS
	// Sessions Management
	GetUserSessions(ctx context.Context, userID pgtype.UUID) ([]repository.GetUserSessionsRow, error)
	DeleteSession(ctx context.Context, arg repository.DeleteSessionParams) error
	DeleteAllSessionsExcept(ctx context.Context, arg repository.DeleteAllSessionsExceptParams) error
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

func (s *AuthService) ResendVerificationEmail(ctx context.Context, email string) error {
	// 1. Fetch the user
	user, err := s.store.GetUserByEmail(ctx, email)
	if err != nil {
		// Security: If the user doesn't exist, return nil to prevent email enumeration
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return apperr.NewInternal("System error", apperr.WithErr(err))
	}

	// 2. Ensure they actually need verification
	if user.VerifiedAt.Valid {
		return apperr.NewConflict("This account is already verified.")
	}

	// 3. Enforce the Cooldown (e.g., 60 seconds)
	if user.LastVerificationSentAt.Valid && time.Since(user.LastVerificationSentAt.Time) < 60*time.Second {
		return apperr.NewTooManyRequests("Please wait a minute before requesting another verification email.")
	}

	// 4. Generate a fresh verification token
	userID := uuid.UUID(user.ID.Bytes)
	tokenStr, err := token.GenerateJWT(userID, s.tokenConfig.VerificationSecret, 24*time.Hour)
	if err != nil {
		return err
	}

	// 5. Execute Transaction: Update throttle timestamp AND queue the outbox event
	return s.store.ExecTx(ctx, func(qtx *repository.Queries) error {
		if err := qtx.UpdateUserLastVerificationSent(ctx, user.ID); err != nil {
			return err
		}

		jsonBytes, _ := json.Marshal(map[string]string{
			"email":    user.Email,
			"username": user.Username,
			"token":    tokenStr,
		})

		_, err = qtx.CreateOutboxEvent(ctx, repository.CreateOutboxEventParams{
			EventType: "user.verify_email", // New event type
			Payload:   jsonBytes,
		})
		return err
	})
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

// GenerateTOTP creates a temporary 2FA secret for the user.
// It returns the raw secret string and an otpauth:// URL for QR code generation.
func (s *AuthService) GenerateTOTP(ctx context.Context, email string) (string, string, error) {
	// The issuer will appear as the app name in Google Authenticator/Authy
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "Bonfire",
		AccountName: email,
		SecretSize:  20,
	})
	if err != nil {
		return "", "", apperr.NewInternal("Failed to generate 2FA secret.", apperr.WithErr(err))
	}

	// Return the secret (to pass back during confirmation) and the URL (for the QR code)
	return key.Secret(), key.URL(), nil
}

// EnableTOTP validates the user's first 6-digit code and permanently saves the secret.
func (s *AuthService) EnableTOTP(ctx context.Context, userID uuid.UUID, secret string, code string) error {
	// 1. Verify the provided 6-digit code against the pending secret
	valid := totp.Validate(code, secret)
	if !valid {
		return apperr.NewUnauthenticated("Invalid authenticator code. Please try again.")
	}

	// 2. Convert standard UUID to pgtype.UUID for sqlc
	var pgUserID pgtype.UUID
	pgUserID.Bytes = userID
	pgUserID.Valid = true

	// 3. Convert string to pgtype.Text for the nullable database column
	var pgSecret pgtype.Text
	pgSecret.String = secret
	pgSecret.Valid = true

	// 4. Save the secret and flip is_totp_enabled to TRUE in the database
	err := s.store.EnableUserTOTP(ctx, repository.EnableUserTOTPParams{
		TotpSecret: pgSecret,
		ID:         pgUserID,
	})
	if err != nil {
		return apperr.NewInternal("Failed to enable 2FA on your account.", apperr.WithErr(err))
	}

	return nil
}

// VerifyLogin2FA validates the TOTP code and completes the login process.
func (s *AuthService) VerifyLogin2FA(ctx context.Context, mfaToken string, code string, userAgent, clientIP string) (map[string]string, error) {
	// 1. Verify the temporary MFA token.
	// NOTE: Replace this with your actual JWT verification logic.
	// You need to extract the userID from the token claims here.
	userID, err := s.ValidateMFAToken(mfaToken)
	if err != nil {
		return nil, apperr.NewUnauthenticated("Invalid or expired MFA token. Please log in again.")
	}

	// Convert to pgtype.UUID for sqlc
	var pgUserID pgtype.UUID
	pgUserID.Bytes = userID
	pgUserID.Valid = true

	// 2. Fetch the user's secret from the database
	totpSecret, err := s.store.GetUserTOTPSecret(ctx, pgUserID)
	if err != nil || !totpSecret.Valid {
		return nil, apperr.NewInternal("Failed to retrieve 2FA configuration.")
	}

	// 3. Validate the 6-digit code mathematically against the secret
	valid := totp.Validate(code, totpSecret.String)
	if !valid {
		return nil, apperr.NewUnauthenticated("Invalid 2FA code.")
	}

	// 4. Code is valid! Generate the real Access and Refresh tokens
	// NOTE: Use the exact same token generation logic you use in your normal Login service.
	accessToken, err := s.generateAccessToken(userID)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.generateRefreshToken(userID)
	if err != nil {
		return nil, err
	}

	// 5. Create the database session
	// NOTE: Adjust expires_at based on your refresh token lifespan (e.g., 7 days)
	_, err = s.store.CreateSession(ctx, repository.CreateSessionParams{
		UserID:       pgUserID,
		RefreshToken: refreshToken,
		UserAgent:    userAgent,
		ClientIp:     clientIP,
		IsBlocked:    false,
		ExpiresAt:    pgtype.Timestamptz{Time: time.Now().Add(7 * 24 * time.Hour), Valid: true},
	})
	if err != nil {
		return nil, apperr.NewInternal("Failed to create user session.")
	}

	// 6. Return the tokens to the handler
	return map[string]string{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	}, nil
}

func (s *AuthService) GetUserByID(ctx context.Context, userID uuid.UUID) (repository.GetUserByIDRow, error) {
	var pgUserID pgtype.UUID
	pgUserID.Bytes = userID
	pgUserID.Valid = true

	return s.store.GetUserByID(ctx, pgUserID)
}

// --- Token Helpers ---

func (s *AuthService) ValidateMFAToken(tokenStr string) (uuid.UUID, error) {
	// Replace "MFATokenSecret" with your actual secret key used during the initial login step
	claims, err := token.VerifyJWT(tokenStr, "YOUR_MFA_TOKEN_SECRET_HERE")
	if err != nil {
		return uuid.Nil, err
	}
	return claims.UserID, nil
}

func (s *AuthService) generateAccessToken(userID uuid.UUID) (string, error) {
	return token.GenerateJWT(userID, s.tokenConfig.AccessSecret, 15*time.Minute)
}

func (s *AuthService) generateRefreshToken(userID uuid.UUID) (string, error) {
	return token.GenerateJWT(userID, s.tokenConfig.RefreshSecret, 7*24*time.Hour)
}

func (s *AuthService) RevokeAllOtherSessions(ctx context.Context, userID uuid.UUID, currentSessionID uuid.UUID) error {
	// You'll need to add DeleteAllSessionsExcept to your interface
	return s.store.DeleteAllSessionsExcept(ctx, repository.DeleteAllSessionsExceptParams{
		UserID: pgtype.UUID{Bytes: userID, Valid: true},
		ID:     pgtype.UUID{Bytes: currentSessionID, Valid: true},
	})
}

// Simple parser to make User-Agents human-readable (like Discord does).
// For production, consider using "github.com/mssola/useragent".
func parseDeviceName(userAgent string) string {
	ua := strings.ToLower(userAgent)
	os := "Unknown OS"
	browser := "Unknown Browser"

	if strings.Contains(ua, "windows") {
		os = "Windows"
	} else if strings.Contains(ua, "mac os") {
		os = "macOS"
	} else if strings.Contains(ua, "linux") {
		os = "Linux"
	} else if strings.Contains(ua, "android") {
		os = "Android"
	} else if strings.Contains(ua, "iphone") || strings.Contains(ua, "ipad") {
		os = "iOS"
	}

	if strings.Contains(ua, "bonfire-client") { // Assuming you have a desktop client
		browser = "Bonfire Client"
	} else if strings.Contains(ua, "firefox") {
		browser = "Firefox"
	} else if strings.Contains(ua, "chrome") {
		browser = "Chrome"
	} else if strings.Contains(ua, "safari") {
		browser = "Safari"
	} else if strings.Contains(ua, "edg") {
		browser = "Edge"
	}

	return os + " (" + browser + ")"
}

// GetDevices retrieves all active sessions and flags the current one based on the refresh token.
func (s *AuthService) GetDevices(ctx context.Context, userID uuid.UUID, currentRefreshToken string) ([]DeviceResponse, error) {
	pgUserID := pgtype.UUID{Bytes: userID, Valid: true}

	sessions, err := s.store.GetUserSessions(ctx, pgUserID)
	if err != nil {
		return nil, apperr.NewInternal("Failed to fetch active devices.", apperr.WithErr(err))
	}

	var devices []DeviceResponse
	for _, sess := range sessions {
		// IMPORTANT: To get "City, Province", you would map sess.ClientIp
		// against a GeoIP database like MaxMind here.
		location := "Unknown Location"
		if sess.ClientIp == "127.0.0.1" || sess.ClientIp == "::1" {
			location = "Localhost"
		}

		devices = append(devices, DeviceResponse{
			ID:        uuid.UUID(sess.ID.Bytes).String(),
			Name:      parseDeviceName(sess.UserAgent),
			Location:  location,
			IPAddress: sess.ClientIp,
			IsCurrent: sess.RefreshToken == currentRefreshToken,
			LastSeen:  sess.LastSeenAt.Time, // Defaults to creation time unless updated on refresh
		})
	}

	return devices, nil
}

// RevokeDevice deletes a specific session, ensuring it belongs to the authenticated user.
func (s *AuthService) RevokeDevice(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID) error {
	err := s.store.DeleteSession(ctx, repository.DeleteSessionParams{
		ID:     pgtype.UUID{Bytes: sessionID, Valid: true},
		UserID: pgtype.UUID{Bytes: userID, Valid: true},
	})
	if err != nil {
		return apperr.NewInternal("Failed to log out of device.", apperr.WithErr(err))
	}
	return nil
}

// RevokeAllOtherDevices deletes all sessions except the one associated with the provided refresh token.
func (s *AuthService) RevokeAllOtherDevices(ctx context.Context, userID uuid.UUID, currentRefreshToken string) error {
	// 1. Fetch the current session to get its ID
	currentSession, err := s.store.GetSession(ctx, currentRefreshToken)
	if err != nil {
		return apperr.NewUnauthenticated("Current session invalid or already logged out.")
	}

	// 2. Delete everything else
	err = s.store.DeleteAllSessionsExcept(ctx, repository.DeleteAllSessionsExceptParams{
		UserID: pgtype.UUID{Bytes: userID, Valid: true},
		ID:     currentSession.ID,
	})
	if err != nil {
		return apperr.NewInternal("Failed to log out of other devices.", apperr.WithErr(err))
	}

	return nil
}
