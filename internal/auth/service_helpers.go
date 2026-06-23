package auth

import (
	"time"

	"github.com/google/uuid"
)

func (s *Service) generateAccessToken(userID uuid.UUID, role string, isVerified bool) (string, error) {
	// Scalable payload mapping
	customClaims := map[string]any{
		"role": role,
		"ver":  isVerified,
	}

	return s.tokenManager.GenerateJWT(userID, s.tokenConfig.AccessSecret, 15*time.Minute, customClaims)
}

func (s *Service) generateRefreshToken(userID uuid.UUID, sessionID uuid.UUID) (string, error) {
	customClaims := map[string]any{
		"sid": sessionID.String(),
	}

	return s.tokenManager.GenerateJWT(userID, s.tokenConfig.RefreshSecret, 7*24*time.Hour, customClaims)
}

func (s *Service) generatePasswordResetToken(userID uuid.UUID) (string, error) {
	customClaims := map[string]any{}

	return s.tokenManager.GenerateJWT(
		userID,
		s.tokenConfig.PasswordResetSecret,
		15*time.Minute,
		customClaims,
	)
}
