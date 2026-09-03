package auth

import (
	"context"
	"errors"

	"bonfire-api/internal/fields"

	"github.com/google/uuid"
)

func (s *Service) PrintWSTicket(ctx context.Context, rawUserID, rawSessionID uuid.UUID) (fields.ID, error) {
	userID, err := fields.ParseRequiredID("user_id", rawUserID)
	if err != nil {
		return fields.ID{}, err
	}

	sessionID, err := fields.ParseRequiredID("session_id", rawSessionID)
	if err != nil {
		return fields.ID{}, err
	}

	sess, err := s.sessionCache.Get(ctx, sessionID)
	if err != nil {
		return fields.ID{}, err
	}

	if sess.IsRevoked() || sess.IsExpired(fields.Now()) || !sess.UserID().Equals(userID) {
		return fields.ID{}, errors.New("Session invalid!")
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
