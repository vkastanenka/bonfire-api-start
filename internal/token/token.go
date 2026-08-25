package token

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"bonfire-api/internal/fields"

	"github.com/golang-jwt/jwt/v5"
)

const (
	DefaultAccessTTL        = 15 * time.Minute
	DefaultRefreshTTL       = 7 * 24 * time.Hour
	DefaultEmailVerifyTTL   = 24 * time.Hour
	DefaultPasswordResetTTL = 15 * time.Minute
	DefaultClockLeeway      = 5 * time.Second
)

type Claims struct {
	UserID    fields.ID `json:"uid"`
	SessionID fields.ID `json:"sid"`
	TokenType Type      `json:"type"`
	jwt.RegisteredClaims
}

func (c Claims) MarshalJSON() ([]byte, error) {
	type Alias Claims
	return json.Marshal(&struct {
		Alias
		TokenType string `json:"type"`
	}{
		Alias:     Alias(c),
		TokenType: c.TokenType.String(),
	})
}

func (c *Claims) UnmarshalJSON(data []byte) error {
	type Alias Claims
	aux := &struct {
		*Alias
		TokenType string `json:"type"`
	}{
		Alias: (*Alias)(c),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	tokenType, err := ParseTypeString(aux.TokenType)
	if err != nil {
		return err
	}
	c.TokenType = tokenType

	return nil
}

type TypeConfig struct {
	Secret string
	TTL    time.Duration
}

type Config struct {
	Issuer        string
	Access        TypeConfig
	Refresh       TypeConfig
	EmailVerify   TypeConfig
	PasswordReset TypeConfig
}

type typeConfig struct {
	secret []byte
	ttl    time.Duration
}

type Provider struct {
	issuer   string
	variants map[TypeValue]typeConfig
}

func NewProvider(cfg Config) (*Provider, error) {
	if cfg.Issuer == "" {
		cfg.Issuer = "bonfire-api"
	}

	specs := map[TypeValue]TypeConfig{
		TypeAccess:        cfg.Access,
		TypeRefresh:       cfg.Refresh,
		TypeEmailVerify:   cfg.EmailVerify,
		TypePasswordReset: cfg.PasswordReset,
	}

	defaults := map[TypeValue]time.Duration{
		TypeAccess:        DefaultAccessTTL,
		TypeRefresh:       DefaultRefreshTTL,
		TypeEmailVerify:   DefaultEmailVerifyTTL,
		TypePasswordReset: DefaultPasswordResetTTL,
	}

	variants := make(map[TypeValue]typeConfig, len(specs))

	for val, spec := range specs {
		t := NewType(val)
		if spec.Secret == "" {
			return nil, fmt.Errorf("token provider initialization failed: secret for %q cannot be empty", t.String())
		}

		ttl := spec.TTL
		if ttl <= 0 {
			ttl = defaults[val]
		}

		variants[val] = typeConfig{
			secret: []byte(spec.Secret),
			ttl:    ttl,
		}
	}

	return &Provider{
		issuer:   cfg.Issuer,
		variants: variants,
	}, nil
}

func (p *Provider) generate(tokenType Type, claims Claims) (string, time.Time, error) {
	spec, exists := p.variants[tokenType.Value()]
	if !exists || len(spec.secret) == 0 {
		return "", time.Time{}, fmt.Errorf("%w: missing signing configuration for type %s", ErrInternal, tokenType.String())
	}

	now := time.Now()
	expiresAt := now.Add(spec.ttl)

	id, err := fields.NewID()
	if err != nil {
		return "", time.Time{}, err
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
		return "", time.Time{}, fmt.Errorf("%w: signing failed: %v", ErrInternal, err)
	}

	return signedToken, expiresAt, nil
}

func (p *Provider) verify(tokenType Type, tokenStr string) (*Claims, error) {
	spec, exists := p.variants[tokenType.Value()]
	if !exists || len(spec.secret) == 0 {
		return nil, fmt.Errorf("%w: missing verification configuration for type %s", ErrInternal, tokenType.String())
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

	if !claims.TokenType.Is(tokenType.Value()) {
		return nil, fmt.Errorf("%w: expected %q token context, got %q", ErrVariantMismatch, tokenType.String(), claims.TokenType.String())
	}

	return claims, nil
}
