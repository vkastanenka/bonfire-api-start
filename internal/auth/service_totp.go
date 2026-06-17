package auth

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/repository"
	"bonfire-api/internal/token"
	"context"
	"net/netip"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pquerna/otp/totp"
)

// GenerateTOTP creates a temporary 2FA secret for the user by looking up their profile.
// It returns the raw secret string and an otpauth:// URL for QR code generation.
func (s *AuthService) GenerateTOTP(ctx context.Context, userID uuid.UUID) (string, string, error) {
	pgUserID := pgtype.UUID{Bytes: userID, Valid: true}

	// 1. Look up the user record using the injected cross-domain provider
	user, err := s.store.UserGet(ctx, pgUserID)
	if err != nil {
		return "", "", apperr.New(apperr.CodeInternal,
			"Failed to retrieve user information for 2FA configuration.",
			apperr.WithErr(err),
		)
	}

	// 2. The issuer will appear as the app name in Google Authenticator/Authy
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "Bonfire",
		AccountName: user.Email, // Safely extracted from the database row
		SecretSize:  20,
	})
	if err != nil {
		return "", "", apperr.New(apperr.CodeInternal, "Failed to generate 2FA secret.", apperr.WithErr(err))
	}

	// Return the secret (to pass back during confirmation) and the URL (for the QR code)
	return key.Secret(), key.URL(), nil
}

// EnableTOTP validates the user's first 6-digit code and permanently saves the secret.
func (s *AuthService) EnableTOTP(ctx context.Context, userID uuid.UUID, secret string, code string) error {
	// 1. Verify the provided 6-digit code against the pending secret
	valid := totp.Validate(code, secret)
	if !valid {
		return apperr.New(apperr.CodeUnauthenticated, "Invalid authenticator code. Please try again.")
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
	err := s.store.UserEnableTOTP(ctx, repository.UserEnableTOTPParams{
		TotpSecret: pgSecret,
		ID:         pgUserID,
	})
	if err != nil {
		return apperr.New(apperr.CodeInternal, "Failed to enable 2FA on your account.", apperr.WithErr(err))
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
		return nil, apperr.New(apperr.CodeUnauthenticated, "Invalid or expired MFA token. Please log in again.")
	}

	// Convert to pgtype.UUID for sqlc
	var pgUserID pgtype.UUID
	pgUserID.Bytes = userID
	pgUserID.Valid = true

	// 2. Fetch the user's secret from the database
	totpSecret, err := s.store.UserGetTOTPSecret(ctx, pgUserID)
	if err != nil || !totpSecret.Valid {
		return nil, apperr.New(apperr.CodeInternal, "Failed to retrieve 2FA configuration.")
	}

	// 3. Validate the 6-digit code mathematically against the secret
	valid := totp.Validate(code, totpSecret.String)
	if !valid {
		return nil, apperr.New(apperr.CodeUnauthenticated, "Invalid 2FA code.")
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

	ipAddr, err := netip.ParseAddr(clientIP)
	if err != nil {
		return nil, apperr.New(apperr.CodeInvalidInput, "Invalid IP address format.", apperr.WithErr(err))
	}

	// 5. Create the database session
	// NOTE: Adjust expires_at based on your refresh token lifespan (e.g., 7 days)
	_, err = s.store.UserSessionCreate(ctx, repository.UserSessionCreateParams{
		UserID:       pgUserID,
		RefreshToken: refreshToken,
		UserAgent:    userAgent,
		ClientIp:     ipAddr,
		IsBlocked:    false,
		ExpiresAt:    pgtype.Timestamptz{Time: time.Now().Add(7 * 24 * time.Hour), Valid: true},
	})
	if err != nil {
		return nil, apperr.New(apperr.CodeInternal, "Failed to create user session.")
	}

	// 6. Return the tokens to the handler
	return map[string]string{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	}, nil
}

func (s *AuthService) ValidateMFAToken(tokenStr string) (uuid.UUID, error) {
	claims, err := token.VerifyJWT(tokenStr, s.tokenConfig.PasswordMFASecret)
	if err != nil {
		return uuid.Nil, err
	}
	return claims.UserID, nil
}
