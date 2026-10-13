package auth

import (
	"net/netip"
	"time"

	"github.com/google/uuid"
)

type SessionDTO struct {
	ID         uuid.UUID  `json:"id"`
	ClientIP   netip.Addr `json:"clientIp"`
	UserAgent  string     `json:"userAgent"`
	OS         string     `json:"os"`
	Client     string     `json:"client"`
	ExpiresAt  time.Time  `json:"expiresAt"`
	LastSeenAt time.Time  `json:"lastSeenAt"`
	CreatedAt  time.Time  `json:"createdAt"`
}

func NewSessionDTO(s *Session) SessionDTO {
	return SessionDTO{
		ID:         s.ID,
		ClientIP:   s.ClientIP,
		UserAgent:  s.UserAgent,
		OS:         s.OS,
		Client:     s.Client,
		ExpiresAt:  s.ExpiresAt,
		LastSeenAt: s.LastSeenAt,
		CreatedAt:  s.CreatedAt,
	}
}

func NewSessionDTOs(sessions []*Session) []SessionDTO {
	dtos := make([]SessionDTO, len(sessions))
	for i, s := range sessions {
		dtos[i] = NewSessionDTO(s)
	}
	return dtos
}
