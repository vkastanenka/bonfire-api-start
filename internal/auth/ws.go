package auth

import (
	"bonfire-api/internal/appctx"
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

func (s *Service) PrintWSTicket(ctx context.Context) (uuid.UUID, error) {
	claims, err := appctx.GetClaims(ctx)
	if err != nil {
		return uuid.UUID{}, err
	}

	sess, err := s.sessionCache.Get(ctx, claims.SessionID)
	if err != nil {
		return uuid.UUID{}, err
	}

	if sess.IsRevoked() || sess.IsExpired(time.Now()) || sess.UserID != claims.UserID {
		return uuid.UUID{}, errors.New("Session invalid!")
	}

	ticketID, err := uuid.NewV7()
	if err != nil {
		return uuid.UUID{}, err
	}

	if err := s.ticketCache.Print(ctx, ticketID, claims.UserID, claims.SessionID); err != nil {
		return uuid.UUID{}, err
	}

	return ticketID, nil
}
