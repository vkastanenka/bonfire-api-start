package session

import (
	"net/netip"
	"time"

	"github.com/google/uuid"
)

type View struct {
	ID         uuid.UUID  `json:"id"`
	ClientIP   netip.Addr `json:"clientIp"`
	UserAgent  string     `json:"userAgent"`
	OS         string     `json:"os"`
	Client     string     `json:"client"`
	ExpiresAt  time.Time  `json:"expiresAt"`
	LastSeenAt time.Time  `json:"lastSeenAt"`
	CreatedAt  time.Time  `json:"createdAt"`
}

func ParseView(s *Session) View {
	return View{
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

func ParseViews(sessions []*Session) []View {
	views := make([]View, len(sessions))
	for i, s := range sessions {
		views[i] = ParseView(s)
	}
	return views
}
