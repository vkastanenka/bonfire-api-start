package token

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	defaultIssuer      = "api.domain.com"
	defaultClockLeeway = 5 * time.Second
)

type Claims struct {
	UserID    uuid.UUID `json:"uid"`
	SessionID uuid.UUID `json:"sid"`
	TokenType Type      `json:"type"`
	jwt.RegisteredClaims
}

type TypeConfig struct {
	Secret string        `json:"secret"`
	TTL    time.Duration `json:"ttl"`
}

func (tc TypeConfig) validate(tokenType Type) error {
	path := "variants." + tokenType.String()

	if tc.Secret == "" {
		return ErrTokenConfigInvalid().
			Meta("token_type", tokenType.String()).
			Meta("secret", "").
			FieldViolation(path+".secret", "Secret key cannot be empty.", "REQUIRED")
	}

	if tc.TTL <= 0 {
		return ErrTokenConfigInvalid().
			Meta("token_type", tokenType.String()).
			Meta("ttl", tc.TTL.String()).
			FieldViolation(path+".ttl", "TTL must be a positive duration.", "OUT_OF_BOUNDS")
	}

	return nil
}

type Provider struct {
	issuer   string
	variants map[Type]TypeConfig
}

type ProviderConfig struct {
	Issuer        string
	Access        TypeConfig
	Refresh       TypeConfig
	EmailVerify   TypeConfig
	PasswordReset TypeConfig
}

func NewProvider(cfg ProviderConfig) (*Provider, error) {
	if cfg.Issuer == "" {
		cfg.Issuer = defaultIssuer
	}

	specs := map[Type]TypeConfig{
		TypeAccess:        cfg.Access,
		TypeRefresh:       cfg.Refresh,
		TypeEmailVerify:   cfg.EmailVerify,
		TypePasswordReset: cfg.PasswordReset,
	}

	for tokenType, spec := range specs {
		if err := spec.validate(tokenType); err != nil {
			return nil, err
		}
	}

	return &Provider{
		issuer:   cfg.Issuer,
		variants: specs,
	}, nil
}

func (p *Provider) generate(tokenType Type, claims Claims) (string, time.Time, error) {
	spec, exists := p.variants[tokenType]
	if !exists || len(spec.Secret) == 0 {
		return "", time.Time{}, ErrTokenConfigMissing().
			Meta("token_type", tokenType.String())
	}

	now := time.Now().UTC().Truncate(time.Second)
	expiresAt := now.Add(spec.TTL)

	claims.TokenType = tokenType
	claims.RegisteredClaims = jwt.RegisteredClaims{
		ID:        uuid.New().String(),
		IssuedAt:  jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
		Issuer:    p.issuer,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(spec.Secret))
	if err != nil {
		return "", time.Time{}, ErrTokenSignFailed().
			Meta("token_type", tokenType.String()).
			Wrap(err)
	}

	return signedToken, expiresAt, nil
}

func (p *Provider) verify(tokenType Type, tokenStr string) (Claims, error) {
	spec, exists := p.variants[tokenType]
	if !exists || spec.Secret == "" {
		return Claims{}, ErrTokenConfigMissing().
			Meta("token_type", tokenType.String())
	}

	token, err := jwt.ParseWithClaims(
		tokenStr,
		Claims{},
		func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing algorithm: %v", t.Header["alg"])
			}
			return []byte(spec.Secret), nil
		},
		jwt.WithIssuer(p.issuer),
		jwt.WithLeeway(defaultClockLeeway),
	)

	if err != nil {
		switch {
		case errors.Is(err, jwt.ErrTokenExpired):
			return Claims{}, ErrTokenExpired().Wrap(err)
		case errors.Is(err, jwt.ErrTokenInvalidIssuer):
			return Claims{}, ErrTokenIssuerInvalid().Wrap(err)
		case errors.Is(err, jwt.ErrTokenMalformed):
			return Claims{}, ErrTokenMalformed().Wrap(err)
		case errors.Is(err, jwt.ErrTokenSignatureInvalid):
			return Claims{}, ErrTokenSignatureInvalid().Wrap(err)
		default:
			return Claims{}, ErrTokenInvalid().Wrap(err)
		}
	}

	parsedClaims, ok := token.Claims.(Claims)
	if !ok || !token.Valid {
		return Claims{}, ErrTokenClaimsInvalid()
	}

	if parsedClaims.TokenType != tokenType {
		return Claims{}, ErrTokenTypeInvalid().
			Meta("expected_type", tokenType.String()).
			Meta("actual_type", parsedClaims.TokenType.String())
	}

	return parsedClaims, nil
}
