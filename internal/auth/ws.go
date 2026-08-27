package auth

import (
	"context"

	"bonfire-api/internal/fields"

	"github.com/google/uuid"
)

// PrintWSTicket generates a single-use, short-lived ticket for establishing a WebSocket connection.
func (s *Service) PrintWSTicket(ctx context.Context, rawUserID uuid.UUID) (fields.ID, error) {
	userID, err := fields.ParseRequiredID("id", rawUserID)
	if err != nil {
		return fields.ID{}, err
	}

	ticketID, err := fields.NewID()
	if err != nil {
		return fields.ID{}, err
	}

	if err := s.ticketCache.Print(ctx, ticketID, userID); err != nil {
		return fields.ID{}, err
	}

	return ticketID, nil
}
