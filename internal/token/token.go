package token

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Manager defines the contract for token operations, accepting arbitrary custom claims
type Manager interface {
	VerifyJWT(tokenString string, secret string) (*Claims, error)
	GenerateJWT(userID uuid.UUID, secret string, duration time.Duration, customClaims map[string]any) (string, error)
	IssueNewBundle(userID uuid.UUID, role string, isVerified bool) (TokenBundle, error)
}

// Claims defines the JWT payload structure.
type Claims struct {
	jwt.RegisteredClaims
	UserID uuid.UUID      `json:"user_id"`
	Flags  int64          `json:"flags,omitempty"`
	Extra  map[string]any `json:"extra,omitempty"` // Allows adding any fields dynamically
}

// Helper methods to make grabbing custom data out of validated claims effortless
func (c *Claims) GetString(key string) string {
	if c.Extra == nil {
		return ""
	}
	val, ok := c.Extra[key].(string)
	if !ok {
		return ""
	}
	return val
}

func (c *Claims) GetBool(key string) bool {
	if c.Extra == nil {
		return false
	}
	val, ok := c.Extra[key].(bool)
	if !ok {
		return false
	}
	return val
}

type JWTManager struct{}

func NewJWTManager() *JWTManager {
	return &JWTManager{}
}

// GenerateJWT creates a new token accepting a flexible customClaims map
func (m *JWTManager) GenerateJWT(userID uuid.UUID, secretKey string, duration time.Duration, customClaims map[string]any) (string, error) {
	claims := Claims{
		UserID: userID,
		Extra:  customClaims, // Pass your scalable map directly here
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        uuid.NewString(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secretKey))
}

func (m *JWTManager) VerifyJWT(tokenString string, secretKey string) (*Claims, error) {
	keyFunc := func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secretKey), nil
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, keyFunc)
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token format or expired")
	}

	return claims, nil
}

// Define constants here so only the token package knows the internal claim keys
const (
	ClaimRole = "role"
	ClaimVer  = "ver"
	ClaimSID  = "sid"
)

// GenerateAccessToken abstracts the claim mapping away from Auth
func (m *JWTManager) GenerateAccessToken(userID uuid.UUID, secret string, role string, isVerified bool) (string, error) {
	claims := map[string]any{
		ClaimRole: role,
		ClaimVer:  isVerified,
	}
	return m.GenerateJWT(userID, secret, 15*time.Minute, claims)
}

// GenerateRefreshToken abstracts the session ID mapping
func (m *JWTManager) GenerateRefreshToken(userID uuid.UUID, secret string, sessionID uuid.UUID) (string, error) {
	claims := map[string]any{
		ClaimSID: sessionID.String(),
	}
	return m.GenerateJWT(userID, secret, 7*24*time.Hour, claims)
}
