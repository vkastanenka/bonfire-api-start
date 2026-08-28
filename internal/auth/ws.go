package auth

import (
	"context"

	"bonfire-api/internal/fields"

	"github.com/google/uuid"
)

// PrintWSTicket generates a single-use, short-lived ticket for establishing a WebSocket connection.
func (s *Service) PrintWSTicket(ctx context.Context, rawUserID, rawSessionID uuid.UUID) (fields.ID, error) {
	userID, err := fields.ParseRequiredID("user_id", rawUserID)
	if err != nil {
		return fields.ID{}, err
	}

	sessionID, err := fields.ParseRequiredID("session_id", rawSessionID)
	if err != nil {
		return fields.ID{}, err
	}

	ticketID, err := fields.NewID()
	if err != nil {
		return fields.ID{}, err
	}

	if err := s.ticketCache.Print(ctx, ticketID, userID, sessionID); err != nil {
		return fields.ID{}, err
	}

	return ticketID, nil
}
