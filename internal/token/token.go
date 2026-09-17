package token

import (
	"bonfire-api/internal/pkg/errs"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	DefaultIssuer      = "bonfire-api"
	DefaultClockLeeway = 5 * time.Second
)

type Claims struct {
	UserID    uuid.UUID `json:"uid"`
	SessionID uuid.UUID `json:"sid"`
	TokenType Type      `json:"type"`
	jwt.RegisteredClaims
}

type Config struct {
	Issuer        string
	Access        TypeConfig
	Refresh       TypeConfig
	EmailVerify   TypeConfig
	PasswordReset TypeConfig
}

type TypeConfig struct {
	Secret string
	TTL    time.Duration
}

type typeConfig struct {
	secret []byte
	ttl    time.Duration
}

type Provider struct {
	issuer   string
	variants map[Type]typeConfig
}

func NewProvider(cfg Config) (*Provider, error) {
	if cfg.Issuer == "" {
		cfg.Issuer = DefaultIssuer
	}

	specs := map[Type]TypeConfig{
		TypeAccess:        cfg.Access,
		TypeRefresh:       cfg.Refresh,
		TypeEmailVerify:   cfg.EmailVerify,
		TypePasswordReset: cfg.PasswordReset,
	}

	variants := make(map[Type]typeConfig, len(specs))

	for val, spec := range specs {
		if spec.Secret == "" {
			return nil, errs.InvalidArgument("invalid token configuration").
				Reason("INVALID_TOKEN_CONFIG").
				Meta("token_type", val.String()).
				Meta("provided_secret", "").
				FieldViolation(
					"variants."+val.String()+".secret",
					"secret key cannot be empty",
					"REQUIRED",
				)
		}

		if spec.TTL <= 0 {
			return nil, errs.InvalidArgument("invalid token configuration").
				Reason("INVALID_TOKEN_CONFIG").
				Meta("token_type", val.String()).
				Meta("provided_ttl", spec.TTL.String()).
				FieldViolation(
					"variants."+val.String()+".ttl",
					"TTL must be a positive duration",
					"OUT_OF_BOUNDS",
				)
		}

		variants[val] = typeConfig{
			secret: []byte(spec.Secret),
			ttl:    spec.TTL,
		}
	}

	return &Provider{
		issuer:   cfg.Issuer,
		variants: variants,
	}, nil
}

func (p *Provider) generate(tokenType Type, claims Claims) (string, time.Time, error) {
	spec, exists := p.variants[tokenType]
	if !exists || len(spec.secret) == 0 {
		return "", time.Time{}, errs.Internal("missing signing configuration for token type").
			Reason("UNCONFIGURED_TOKEN_TYPE").
			Meta("token_type", tokenType.String())
	}

	now := time.Now().UTC().Truncate(time.Second)
	expiresAt := now.Add(spec.ttl)

	id, err := uuid.NewV7()
	if err != nil {
		return "", time.Time{}, errs.Internal("failed to generate token uuid").
			Reason("UUID_GENERATION_FAILED").
			Wrap(err)
	}

	claims.TokenType = tokenType
	claims.RegisteredClaims = jwt.RegisteredClaims{
		ID:        id.String(),
		IssuedAt:  jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
		Issuer:    p.issuer,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(spec.secret)
	if err != nil {
		return "", time.Time{}, errs.Internal("failed to sign token").
			Reason("TOKEN_SIGNING_FAILED").
			Meta("token_type", tokenType.String()).
			Wrap(err)
	}

	return signedToken, expiresAt, nil
}

func (p *Provider) verify(tokenType Type, tokenStr string) (*Claims, error) {
	spec, exists := p.variants[tokenType]
	if !exists || len(spec.secret) == 0 {
		return nil, errs.Internal("missing verification configuration for token type").
			Reason("UNCONFIGURED_TOKEN_TYPE").
			Meta("token_type", tokenType.String())
	}

	token, err := jwt.ParseWithClaims(
		tokenStr,
		&Claims{},
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				algVal := "none"
				if alg, ok := t.Header["alg"].(string); ok {
					algVal = alg
				}
				return nil, fmt.Errorf("unexpected signing algorithm: %s", algVal)
			}
			return spec.secret, nil
		},
		jwt.WithIssuer(p.issuer),
		jwt.WithLeeway(DefaultClockLeeway),
	)

	if err != nil {
		switch {
		case errors.Is(err, jwt.ErrTokenExpired):
			return nil, errs.Unauthenticated("token has expired").
				Reason("TOKEN_EXPIRED").
				Wrap(err)
		case errors.Is(err, jwt.ErrTokenInvalidIssuer):
			return nil, errs.Unauthenticated("token issuer mismatch").
				Reason("TOKEN_ISSUER_MISMATCH").
				Meta("expected_issuer", p.issuer).
				Wrap(err)
		case errors.Is(err, jwt.ErrTokenMalformed):
			return nil, errs.InvalidArgument("malformed token format").
				Reason("TOKEN_MALFORMED").
				Meta("provided_token", tokenStr).
				FieldViolation("token", "provided JWT string is malformed or unparseable", "INVALID_FORMAT").
				Wrap(err)
		case errors.Is(err, jwt.ErrTokenSignatureInvalid):
			return nil, errs.Unauthenticated("invalid token signature").
				Reason("TOKEN_SIGNATURE_INVALID").
				Wrap(err)
		default:
			return nil, errs.Unauthenticated("invalid token").
				Reason("TOKEN_INVALID").
				Wrap(err)
		}
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errs.Unauthenticated("invalid or corrupt token claims").
			Reason("TOKEN_CLAIMS_INVALID")
	}

	if claims.TokenType != tokenType {
		return nil, errs.Unauthenticated("token variant mismatch").
			Reason("TOKEN_VARIANT_MISMATCH").
			Meta("expected_type", tokenType.String()).
			Meta("actual_type", claims.TokenType.String())
	}

	return claims, nil
}
