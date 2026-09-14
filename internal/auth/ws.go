package auth

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

func (s *Service) PrintWSTicket(ctx context.Context, userID, sessionID uuid.UUID) (uuid.UUID, error) {
	sess, err := s.sessionCache.Get(ctx, sessionID)
	if err != nil {
		return uuid.UUID{}, err
	}

	if sess.IsRevoked() || sess.IsExpired(time.Now()) || sess.UserID != userID {
		return uuid.UUID{}, errors.New("Session invalid!")
	}

	ticketID, err := uuid.NewV7()
	if err != nil {
		return uuid.UUID{}, err
	}

	if err := s.ticketCache.Print(ctx, ticketID, userID, sessionID); err != nil {
		return uuid.UUID{}, err
	}

	return ticketID, nil
}
