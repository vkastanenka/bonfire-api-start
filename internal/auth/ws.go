package auth

import (
	"context"
	"errors"
	"time"

	"bonfire-api/internal/apperr"

	"github.com/google/uuid"
)

// WSTicketTTL is the single-use lifespan of a WebSocket handshake ticket.
const WSTicketTTL = 20 * time.Second

// WSTicket generates a single-use, short-lived ticket for establishing a WebSocket connection.
func (s *Service) WSTicket(ctx context.Context, uid uuid.UUID) (uuid.UUID, error) {
	// 1. Guard Input
	if uid == uuid.Nil {
		return uuid.Nil, apperr.NewInvalidArgument(
			errors.New("user ID cannot be nil"),
			apperr.WithMsg("Invalid user ID"),
		)
	}

	// 2. Generate UUIDv7 Ticket ID
	ticketID, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, apperr.NewInternal(err, apperr.WithMsg("Failed to generate websocket ticket"))
	}

	// 3. Persist Ticket in Ephemeral Store (Redis/Key-Value)
	if err := s.tickets.SetTicket(ctx, ticketID, uid, WSTicketTTL); err != nil {
		return uuid.Nil, apperr.NewInternal(err, apperr.WithMsg("Failed to store websocket ticket"))
	}

	return ticketID, nil
}
