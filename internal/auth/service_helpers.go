package auth

import (
	"time"

	"github.com/google/uuid"
)

func (s *AuthService) generateAccessToken(userID uuid.UUID) (string, error) {
	return s.tokenManager.GenerateJWT(userID, s.tokenConfig.AccessSecret, 15*time.Minute)
}

func (s *AuthService) generateRefreshToken(userID uuid.UUID) (string, error) {
	return s.tokenManager.GenerateJWT(userID, s.tokenConfig.RefreshSecret, 7*24*time.Hour)
}
