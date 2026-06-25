package token

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// --- TOKEN CONSTANTS ---

const (
	AccessTokenTTL        = 15 * time.Minute
	RefreshTokenTTL       = 7 * 24 * time.Hour
	VerificationTokenTTL  = 1 * time.Hour
	PasswordResetTokenTTL = 15 * time.Minute
	PasswordMFATokenTTL   = 5 * time.Minute
)

// --- TOKEN TYPES ---

type Pair struct {
	AccessToken  string
	RefreshToken string
}

type Claims struct {
	jwt.RegisteredClaims
	UserID          uuid.UUID `json:"user_id"`
	SecurityVersion int       `json:"security_version"`
	Role            string    `json:"role,omitempty"`
	IsVerified      bool      `json:"ver,omitempty"`
	SessionID       uuid.UUID `json:"sid,omitempty"`
}

type Service struct {
	accessSecret        []byte
	refreshSecret       []byte
	verificationSecret  []byte
	passwordResetSecret []byte
	passwordMFASecret   []byte
}

// --- TOKEN INITIALIZATION ---

func NewService(accessSecret string, refreshSecret string, verificationSecret string, passwordResetSecret string, passwordMFASecret string) *Service {
	return &Service{
		accessSecret:        []byte(accessSecret),
		refreshSecret:       []byte(refreshSecret),
		verificationSecret:  []byte(verificationSecret),
		passwordResetSecret: []byte(passwordResetSecret),
		passwordMFASecret:   []byte(passwordMFASecret),
	}
}

// --- TOKEN METHODS ---

// GenerateTokenPair
func (m *Service) GenerateTokenPair(userID uuid.UUID, role string, isVerified bool, securityVersion int, sessionID uuid.UUID) (Pair, error) {
	accessToken, err := m.GenerateAccessToken(userID, role, isVerified, securityVersion)
	if err != nil {
		return Pair{}, err
	}

	refreshToken, err := m.GenerateRefreshToken(userID, securityVersion, sessionID)
	if err != nil {
		return Pair{}, err
	}

	return Pair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// GenerateAccessToken
func (m *Service) GenerateAccessToken(userID uuid.UUID, role string, isVerified bool, securityVersion int) (string, error) {
	return m.generate(userID, AccessTokenTTL, Claims{
		Role:            role,
		IsVerified:      isVerified,
		SecurityVersion: securityVersion,
	}, m.accessSecret)
}

// GenerateRefreshToken
func (m *Service) GenerateRefreshToken(userID uuid.UUID, securityVersion int, sessionID uuid.UUID) (string, error) {
	return m.generate(userID, RefreshTokenTTL, Claims{
		SessionID:       sessionID,
		SecurityVersion: securityVersion,
	}, m.refreshSecret)
}

func (m *Service) GenerateVerification(userID uuid.UUID, securityVersion int) (string, error) {
	return m.generate(userID, VerificationTokenTTL, Claims{SecurityVersion: securityVersion}, m.verificationSecret)
}

func (m *Service) GeneratePasswordReset(userID uuid.UUID, securityVersion int) (string, error) {
	return m.generate(userID, PasswordResetTokenTTL, Claims{SecurityVersion: securityVersion}, m.passwordResetSecret)
}

func (m *Service) GeneratePasswordMFA(userID uuid.UUID, securityVersion int) (string, error) {
	return m.generate(userID, PasswordMFATokenTTL, Claims{SecurityVersion: securityVersion}, m.passwordMFASecret)
}

func (m *Service) generate(userID uuid.UUID, duration time.Duration, claims Claims, secret []byte) (string, error) {
	claims.UserID = userID
	claims.RegisteredClaims = jwt.RegisteredClaims{
		ID:        uuid.NewString(),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
		Issuer:    "bonfire-api",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

func (m *Service) VerifyAccess(tokenString string) (*Claims, error) {
	return m.verify(tokenString, m.accessSecret)
}

func (m *Service) VerifyRefresh(tokenString string) (*Claims, error) {
	return m.verify(tokenString, m.refreshSecret)
}

func (m *Service) VerifyVerification(tokenString string) (*Claims, error) {
	return m.verify(tokenString, m.verificationSecret)
}

func (m *Service) VerifyPasswordReset(tokenString string) (*Claims, error) {
	return m.verify(tokenString, m.passwordResetSecret)
}

func (m *Service) VerifyPasswordMFA(tokenString string) (*Claims, error) {
	return m.verify(tokenString, m.passwordMFASecret)
}

func (m *Service) verify(tokenString string, secret []byte) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return secret, nil
	})

	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, errors.New("invalid claims format")
	}

	if claims.Issuer != "bonfire-api" {
		return nil, errors.New("invalid issuer")
	}

	return claims, nil
}
