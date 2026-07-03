package token

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Type string

type Claims struct {
	UserID    uuid.UUID `json:"uid"`
	SessionID uuid.UUID `json:"sid,omitempty"`
	Type      Type      `json:"typ"`
	jwt.RegisteredClaims
}

type Config struct {
	AccessSecret  string
	RefreshSecret string
	Issuer        string
}

type Pair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type Manager struct {
	issuer  string
	secrets map[Type][]byte
}

const (
	TypeAccess  Type = "access"
	TypeRefresh Type = "refresh"
)

const (
	AccessTokenTTL  = 15 * time.Minute
	RefreshTokenTTL = 7 * 24 * time.Hour
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

func NewManager(cfg Config) (*Manager, error) {
	if cfg.AccessSecret == "" || cfg.RefreshSecret == "" {
		return nil, fmt.Errorf("token manager initialization failed: critical secrets cannot be empty")
	}

	if cfg.Issuer == "" {
		cfg.Issuer = "bonfire-api"
	}

	return &Manager{
		issuer: cfg.Issuer,
		secrets: map[Type][]byte{
			TypeAccess:  []byte(cfg.AccessSecret),
			TypeRefresh: []byte(cfg.RefreshSecret),
		},
	}, nil
}

func (m *Manager) GenerateTokenPair(userID uuid.UUID, sessionID uuid.UUID) (Pair, error) {
	accessToken, err := m.GenerateAccessToken(userID)
	if err != nil {
		return Pair{}, err
	}

	refreshToken, err := m.GenerateRefreshToken(userID, sessionID)
	if err != nil {
		return Pair{}, err
	}

	return Pair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (m *Manager) GenerateAccessToken(userID uuid.UUID) (string, error) {
	return m.generate(userID, TypeAccess, AccessTokenTTL, Claims{})
}

func (m *Manager) GenerateRefreshToken(userID uuid.UUID, sessionID uuid.UUID) (string, error) {
	return m.generate(userID, TypeRefresh, RefreshTokenTTL, Claims{
		SessionID: sessionID,
	})
}

func (m *Manager) VerifyAccess(tokenStr string) (*Claims, error) {
	return m.verify(tokenStr, TypeAccess)
}

func (m *Manager) VerifyRefresh(tokenStr string) (*Claims, error) {
	return m.verify(tokenStr, TypeRefresh)
}

func (m *Manager) generate(userID uuid.UUID, tokenType Type, ttl time.Duration, claims Claims) (string, error) {
	secret, exists := m.secrets[tokenType]
	if !exists || len(secret) == 0 {
		return "", fmt.Errorf("%w: missing signing key for type %s", ErrInternal, tokenType)
	}

	now := time.Now()
	claims.UserID = userID
	claims.Type = tokenType
	claims.RegisteredClaims = jwt.RegisteredClaims{
		ID:        uuid.NewString(),
		IssuedAt:  jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		Issuer:    m.issuer,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(secret)
	if err != nil {
		return "", fmt.Errorf("%w: signing failed: %v", ErrInternal, err)
	}

	return signedToken, nil
}

func (m *Manager) verify(tokenStr string, expectedType Type) (*Claims, error) {
	secret, exists := m.secrets[expectedType]
	if !exists || len(secret) == 0 {
		return nil, fmt.Errorf("%w: missing verification key for type %s", ErrInternal, expectedType)
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
			return nil, fmt.Errorf("%w: %v", ErrTokenExpired, err)
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

	if claims.Type != expectedType {
		return nil, fmt.Errorf("%w: expected %q token context, got %q", ErrTypeMismatch, expectedType, claims.Type)
	}

	return claims, nil
}
