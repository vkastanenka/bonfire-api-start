package token

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	DefaultAccessTTL        = 15 * time.Minute
	DefaultRefreshTTL       = 7 * 24 * time.Hour
	DefaultEmailVerifyTTL   = 24 * time.Hour
	DefaultPasswordResetTTL = 15 * time.Minute
	DefaultClockLeeway      = 5 * time.Second
)

var (
	ErrTokenExpired          = errors.New("token has expired")
	ErrTokenMalformed        = errors.New("token is malformed")
	ErrTokenSignatureInvalid = errors.New("token signature is invalid")
	ErrTokenInvalid          = errors.New("token is invalid")
	ErrIssuerMismatch        = errors.New("token issuer is invalid")
	ErrVariantMismatch       = errors.New("token type mismatch")
	ErrInternal              = errors.New("internal cryptographic error")
)

type Claims struct {
	UserID    uuid.UUID `json:"uid"`
	SessionID uuid.UUID `json:"sid"`
	Variant   Variant   `json:"var"`
	jwt.RegisteredClaims
}

type Config struct {
	AccessSecret        string
	RefreshSecret       string
	VerifySecret        string
	PasswordResetSecret string
	Issuer              string

	AccessTTL        time.Duration
	RefreshTTL       time.Duration
	EmailVerifyTTL   time.Duration
	PasswordResetTTL time.Duration
}

type Provider struct {
	issuer  string
	secrets map[Variant][]byte
	ttls    map[Variant]time.Duration
}

func NewProvider(cfg Config) (*Provider, error) {
	if cfg.AccessSecret == "" || cfg.RefreshSecret == "" || cfg.VerifySecret == "" || cfg.PasswordResetSecret == "" {
		return nil, fmt.Errorf("token provider initialization failed: critical secrets cannot be empty")
	}

	if cfg.Issuer == "" {
		cfg.Issuer = "bonfire-api"
	}

	accessTTL := cfg.AccessTTL
	if accessTTL <= 0 {
		accessTTL = DefaultAccessTTL
	}

	refreshTTL := cfg.RefreshTTL
	if refreshTTL <= 0 {
		refreshTTL = DefaultRefreshTTL
	}

	emailVerifyTTL := cfg.EmailVerifyTTL
	if emailVerifyTTL <= 0 {
		emailVerifyTTL = DefaultEmailVerifyTTL
	}

	passwordResetTTL := cfg.PasswordResetTTL
	if passwordResetTTL <= 0 {
		passwordResetTTL = DefaultPasswordResetTTL
	}

	return &Provider{
		issuer: cfg.Issuer,
		secrets: map[Variant][]byte{
			VariantAccess:        []byte(cfg.AccessSecret),
			VariantRefresh:       []byte(cfg.RefreshSecret),
			VariantEmailVerify:   []byte(cfg.VerifySecret),
			VariantPasswordReset: []byte(cfg.PasswordResetSecret),
		},
		ttls: map[Variant]time.Duration{
			VariantAccess:        accessTTL,
			VariantRefresh:       refreshTTL,
			VariantEmailVerify:   emailVerifyTTL,
			VariantPasswordReset: passwordResetTTL,
		},
	}, nil
}

func (p *Provider) generate(tokenVariant Variant, claims Claims) (string, time.Time, error) {
	secret, exists := p.secrets[tokenVariant]
	if !exists || len(secret) == 0 {
		return "", time.Time{}, fmt.Errorf("%w: missing signing key for type %s", ErrInternal, tokenVariant)
	}

	ttl, exists := p.ttls[tokenVariant]
	if !exists || ttl <= 0 {
		return "", time.Time{}, fmt.Errorf("%w: missing ttl configuration for type %s", ErrInternal, tokenVariant)
	}

	now := time.Now()
	expiresAt := now.Add(ttl)

	claims.Variant = tokenVariant
	claims.RegisteredClaims = jwt.RegisteredClaims{
		ID:        uuid.NewString(),
		IssuedAt:  jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
		Issuer:    p.issuer,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("%w: signing failed: %v", ErrInternal, err)
	}

	return signedToken, expiresAt, nil
}

func (p *Provider) verify(tokenVariant Variant, tokenStr string) (*Claims, error) {
	secret, exists := p.secrets[tokenVariant]
	if !exists || len(secret) == 0 {
		return nil, fmt.Errorf("%w: missing verification key for type %s", ErrInternal, tokenVariant)
	}

	token, err := jwt.ParseWithClaims(
		tokenStr,
		&Claims{},
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing algorithm variant: %v", t.Header["alg"])
			}
			return secret, nil
		},
		jwt.WithLeeway(DefaultClockLeeway),
	)

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

	if claims.Issuer != p.issuer {
		return nil, fmt.Errorf("%w: expected %q, got %q", ErrIssuerMismatch, p.issuer, claims.Issuer)
	}

	if claims.Variant != tokenVariant {
		return nil, fmt.Errorf("%w: expected %q token context, got %q", ErrVariantMismatch, tokenVariant, claims.Variant)
	}

	return claims, nil
}
