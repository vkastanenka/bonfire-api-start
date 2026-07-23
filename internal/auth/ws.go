package auth

import (
	"context"
	"errors"
	"time"

	"bonfire-api/internal/errs"

	"github.com/google/uuid"
)

// WSTicketTTL is the single-use lifespan of a WebSocket handshake ticket.
const WSTicketTTL = 20 * time.Second

// WSTicket generates a single-use, short-lived ticket for establishing a WebSocket connection.
func (s *Service) WSTicket(ctx context.Context, uid uuid.UUID) (uuid.UUID, error) {
	// 1. Guard Input
	if uid == uuid.Nil {
		return uuid.Nil, errs.InvalidArgument("Invalid user ID.").
			FieldViolation("user_id", "User ID cannot be empty.", "REQUIRED").
			Wrap(errors.New("user ID cannot be nil"))
	}

	// 2. Generate UUIDv7 Ticket ID
	ticketID, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, errs.Internal("failed to generate websocket ticket").Wrap(err)
	}

	// 3. Persist Ticket in Ephemeral Store (Redis/Key-Value)
	if err := s.tickets.SetTicket(ctx, ticketID, uid, WSTicketTTL); err != nil {
		return uuid.Nil, errs.Internal("failed to store websocket ticket").Wrap(err)
	}

	return ticketID, nil
}
