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

type VariantConfig struct {
	Secret string
	TTL    time.Duration
}

type Config struct {
	Issuer        string
	Access        VariantConfig
	Refresh       VariantConfig
	EmailVerify   VariantConfig
	PasswordReset VariantConfig
}

type variantConfig struct {
	secret []byte
	ttl    time.Duration
}

type Provider struct {
	issuer   string
	variants map[Variant]variantConfig
}

func NewProvider(cfg Config) (*Provider, error) {
	if cfg.Issuer == "" {
		cfg.Issuer = "bonfire-api"
	}

	specs := map[Variant]VariantConfig{
		VariantAccess:        cfg.Access,
		VariantRefresh:       cfg.Refresh,
		VariantEmailVerify:   cfg.EmailVerify,
		VariantPasswordReset: cfg.PasswordReset,
	}

	defaults := map[Variant]time.Duration{
		VariantAccess:        DefaultAccessTTL,
		VariantRefresh:       DefaultRefreshTTL,
		VariantEmailVerify:   DefaultEmailVerifyTTL,
		VariantPasswordReset: DefaultPasswordResetTTL,
	}

	variants := make(map[Variant]variantConfig, len(specs))

	for variant, spec := range specs {
		if spec.Secret == "" {
			return nil, fmt.Errorf("token provider initialization failed: secret for %q cannot be empty", variant)
		}

		ttl := spec.TTL
		if ttl <= 0 {
			ttl = defaults[variant]
		}

		variants[variant] = variantConfig{
			secret: []byte(spec.Secret),
			ttl:    ttl,
		}
	}

	return &Provider{
		issuer:   cfg.Issuer,
		variants: variants,
	}, nil
}

func (p *Provider) generate(tokenVariant Variant, claims Claims) (string, time.Time, error) {
	spec, exists := p.variants[tokenVariant]
	if !exists || len(spec.secret) == 0 {
		return "", time.Time{}, fmt.Errorf("%w: missing signing configuration for type %s", ErrInternal, tokenVariant)
	}

	now := time.Now()
	expiresAt := now.Add(spec.ttl)

	claims.Variant = tokenVariant
	claims.RegisteredClaims = jwt.RegisteredClaims{
		ID:        uuid.NewString(),
		IssuedAt:  jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
		Issuer:    p.issuer,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(spec.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("%w: signing failed: %v", ErrInternal, err)
	}

	return signedToken, expiresAt, nil
}

func (p *Provider) verify(tokenVariant Variant, tokenStr string) (*Claims, error) {
	spec, exists := p.variants[tokenVariant]
	if !exists || len(spec.secret) == 0 {
		return nil, fmt.Errorf("%w: missing verification configuration for type %s", ErrInternal, tokenVariant)
	}

	token, err := jwt.ParseWithClaims(
		tokenStr,
		&Claims{},
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing algorithm variant: %v", t.Header["alg"])
			}
			return spec.secret, nil
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
