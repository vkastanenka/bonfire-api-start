package auth

import (
	"bonfire-api/internal/apperr"
	"context"
	"time"

	"github.com/google/uuid"
)

// TODO: Update to config
const WSTicketTTL = 20 * time.Second

type WSTicketData struct {
	UserID uuid.UUID `json:"user_id"`
}

func (s *Service) WSTicket(ctx context.Context, p WSTicketData) (uuid.UUID, error) {
	ticketID, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, apperr.NewInternal(err, apperr.WithMsg("Failed to generate websocket ticket"))
	}

	// ticketKey := cache.WSTicketKey(ticketID)
	// err = s.cache.Set(ctx, ticketKey, p, WSTicketTTL)
	// if err != nil {
	// 	return uuid.Nil, err
	// }

	return ticketID, nil
}
