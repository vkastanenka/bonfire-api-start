package token

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Claims struct {
	UserID    uuid.UUID `json:"uid"`
	SessionID uuid.UUID `json:"sid"`
	Type      Type      `json:"type"`
	jwt.RegisteredClaims
}

type Config struct {
	AccessSecret  string
	RefreshSecret string
	VerifySecret  string
	Issuer        string
}

type Type string

const (
	TypeAccess        Type = "access"
	TypeRefresh       Type = "refresh"
	TypeEmailVerify   Type = "email-verify"
	TypePasswordReset Type = "password-reset"
)

const (
	AccessTTL        = 15 * time.Minute
	RefreshTTL       = 7 * 24 * time.Hour
	EmailVerifyTTL   = 24 * time.Hour
	PasswordResetTTL = 15 * time.Minute
)

var (
	ErrTokenExpired          = errors.New("token has expired")
	ErrTokenMalformed        = errors.New("token is malformed")
	ErrTokenSignatureInvalid = errors.New("token signature is invalid")
	ErrTokenInvalid          = errors.New("token is invalid")
	ErrIssuerMismatch        = errors.New("token issuer is invalid")
	ErrTypeMismatch          = errors.New("token type mismatch")
	ErrInternal              = errors.New("internal cryptographic error")
)

type Manager struct {
	issuer  string
	secrets map[Type][]byte
}

func NewManager(cfg Config) (*Manager, error) {
	if cfg.AccessSecret == "" || cfg.RefreshSecret == "" || cfg.VerifySecret == "" {
		return nil, fmt.Errorf("token manager initialization failed: critical secrets cannot be empty")
	}

	if cfg.Issuer == "" {
		cfg.Issuer = "bonfire-api"
	}

	return &Manager{
		issuer: cfg.Issuer,
		secrets: map[Type][]byte{
			TypeAccess:      []byte(cfg.AccessSecret),
			TypeRefresh:     []byte(cfg.RefreshSecret),
			TypeEmailVerify: []byte(cfg.VerifySecret),
		},
	}, nil
}

type PairParams struct {
	UserID    uuid.UUID
	SessionID uuid.UUID
}

type Pair struct {
	Access           string    `json:"access_token"`
	AccessExpiresAt  time.Time `json:"access_token_expires_at"`
	Refresh          string    `json:"refresh_token"`
	RefreshExpiresAt time.Time `json:"refresh_token_expires_at"`
}

func (m *Manager) GeneratePair(p PairParams) (Pair, error) {
	access, accessExpiresAt, err := m.GenerateAccess(p)
	if err != nil {
		return Pair{}, err
	}

	refresh, refreshExpiresAt, err := m.GenerateRefresh(p)
	if err != nil {
		return Pair{}, err
	}

	return Pair{
		Access:           access,
		AccessExpiresAt:  accessExpiresAt,
		Refresh:          refresh,
		RefreshExpiresAt: refreshExpiresAt,
	}, nil
}

func (m *Manager) GenerateAccess(p PairParams) (string, time.Time, error) {
	return m.generate(TypeAccess, AccessTTL, Claims{
		UserID:    p.UserID,
		SessionID: p.SessionID,
	})
}

func (m *Manager) GenerateRefresh(p PairParams) (string, time.Time, error) {
	return m.generate(TypeRefresh, RefreshTTL, Claims{
		UserID:    p.UserID,
		SessionID: p.SessionID,
	})
}

func (m *Manager) GenerateEmailVerify(userID uuid.UUID) (string, time.Time, error) {
	return m.generate(TypeEmailVerify, EmailVerifyTTL, Claims{
		UserID: userID,
	})
}

func (m *Manager) GeneratePasswordReset(userID uuid.UUID) (string, time.Time, error) {
	return m.generate(TypePasswordReset, PasswordResetTTL, Claims{
		UserID: userID,
	})
}

func (m *Manager) generate(tokenType Type, ttl time.Duration, claims Claims) (string, time.Time, error) {
	secret, exists := m.secrets[tokenType]
	if !exists || len(secret) == 0 {
		return "", time.Time{}, fmt.Errorf("%w: missing signing key for type %s", ErrInternal, tokenType)
	}

	now := time.Now()
	expiresAt := now.Add(ttl)

	claims.Type = tokenType
	claims.RegisteredClaims = jwt.RegisteredClaims{
		ID:        uuid.NewString(),
		IssuedAt:  jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
		Issuer:    m.issuer,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("%w: signing failed: %v", ErrInternal, err)
	}

	return signedToken, expiresAt, nil
}

func (m *Manager) VerifyAccess(tokenStr string) (*Claims, error) {
	return m.verify(TypeAccess, tokenStr)
}

func (m *Manager) VerifyRefresh(tokenStr string) (*Claims, error) {
	return m.verify(TypeRefresh, tokenStr)
}

func (m *Manager) VerifyEmailVerify(tokenStr string) (*Claims, error) {
	return m.verify(TypeEmailVerify, tokenStr)
}

func (m *Manager) VerifyPasswordReset(tokenStr string) (*Claims, error) {
	return m.verify(TypePasswordReset, tokenStr)
}

func (m *Manager) verify(tokenType Type, tokenStr string) (*Claims, error) {
	secret, exists := m.secrets[tokenType]
	if !exists || len(secret) == 0 {
		return nil, fmt.Errorf("%w: missing verification key for type %s", ErrInternal, tokenType)
	}

	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing algorithm variant: %v", t.Header["alg"])
		}
		return secret, nil
	})

	if err != nil {
		switch {
		case errors.Is(err, jwt.ErrTokenExpired):
			return nil, fmt.Errorf("%w (jwt detail: %v)", ErrTokenExpired, err)
		case errors.Is(err, jwt.ErrTokenMalformed):
			return nil, fmt.Errorf("%w: %v", ErrTokenMalformed, err)
		case errors.Is(err, jwt.ErrTokenSignatureInvalid):
			return nil, fmt.Errorf("%w: %v", ErrTokenSignatureInvalid, err)
		default:
			return nil, fmt.Errorf("%w: %v", ErrTokenInvalid, err)
		}
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("%w: claims structure corrupt or invalid", ErrTokenInvalid)
	}

	if claims.Issuer != m.issuer {
		return nil, fmt.Errorf("%w: expected %q, got %q", ErrIssuerMismatch, m.issuer, claims.Issuer)
	}

	if claims.Type != tokenType {
		return nil, fmt.Errorf("%w: expected %q token context, got %q", ErrTypeMismatch, tokenType, claims.Type)
	}

	return claims, nil
}
