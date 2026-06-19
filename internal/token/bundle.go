package token

import uuid "github.com/google/uuid"

type TokenBundle struct {
	AccessToken  string
	RefreshToken string
	SessionID    uuid.UUID
}

// IssueNewBundle is the factory that knows how to build the bundle
func (m *JWTManager) IssueNewBundle(userID uuid.UUID, role string, isVerified bool) (TokenBundle, error) {
	sessionID, err := uuid.NewV7()
	if err != nil {
		return TokenBundle{}, err
	}

	access, err := m.GenerateAccessToken(userID, "YOUR_SECRET", role, isVerified)
	if err != nil {
		return TokenBundle{}, err
	}

	refresh, err := m.GenerateRefreshToken(userID, "YOUR_SECRET", sessionID)
	if err != nil {
		return TokenBundle{}, err
	}

	return TokenBundle{
		AccessToken:  access,
		RefreshToken: refresh,
		SessionID:    sessionID,
	}, nil
}
