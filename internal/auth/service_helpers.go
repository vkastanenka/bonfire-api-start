package auth

import (
	"bonfire-api/internal/token"
	"time"

	"github.com/google/uuid"
)

func (s *AuthService) generateAccessToken(userID uuid.UUID) (string, error) {
	return token.GenerateJWT(userID, s.tokenConfig.AccessSecret, 15*time.Minute)
}

func (s *AuthService) generateRefreshToken(userID uuid.UUID) (string, error) {
	return token.GenerateJWT(userID, s.tokenConfig.RefreshSecret, 7*24*time.Hour)
}
