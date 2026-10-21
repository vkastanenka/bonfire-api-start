package auth

import (
	"net/netip"
	"time"

	"github.com/google/uuid"
)

type SessionDTO struct {
	ID         uuid.UUID  `json:"id"`
	IsCurrent  bool       `json:"isCurrent"`
	ClientIP   netip.Addr `json:"clientIp"`
	UserAgent  string     `json:"userAgent"`
	OS         string     `json:"os"`
	Client     string     `json:"client"`
	ExpiresAt  time.Time  `json:"expiresAt"`
	LastSeenAt time.Time  `json:"lastSeenAt"`
	CreatedAt  time.Time  `json:"createdAt"`
}

func NewSessionDTO(s *Session, currentSessID uuid.UUID) *SessionDTO {
	if s == nil {
		return nil
	}
	return &SessionDTO{
		ID:         s.ID,
		IsCurrent:  s.ID == currentSessID,
		ClientIP:   s.ClientIP,
		UserAgent:  s.UserAgent,
		OS:         s.OS,
		Client:     s.Client,
		ExpiresAt:  s.ExpiresAt,
		LastSeenAt: s.LastSeenAt,
		CreatedAt:  s.CreatedAt,
	}
}

func NewSessionDTOs(sessions []*Session, currentSessID uuid.UUID) []SessionDTO {
	if len(sessions) == 0 {
		return nil
	}

	dtos := make([]SessionDTO, 0, len(sessions))
	for _, s := range sessions {
		if dto := NewSessionDTO(s, currentSessID); dto != nil {
			dtos = append(dtos, *dto)
		}
	}

	return dtos
}
