package store

import (
	"context"
	"fmt"
	"time"

	"bonfire-api/internal/cache"

	"github.com/google/uuid"
)

type TicketStore struct {
	q cache.Querier
}

func NewTicketStore(q cache.Querier) *TicketStore {
	return &TicketStore{q: q}
}

func ticketKey(id uuid.UUID) string {
	return fmt.Sprintf("ticket:{%s}", id.String())
}

func (s *TicketStore) SetTicket(ctx context.Context, ticketID, userID uuid.UUID, ttl time.Duration) error {
	k := ticketKey(ticketID)
	if err := s.q.Set(ctx, k, userID, ttl); err != nil {
		return cache.NewError(err, cache.ScopeTicket)
	}
	return nil
}

func (s *TicketStore) ConsumeTicket(ctx context.Context, ticketID uuid.UUID) (uuid.UUID, error) {
	k := ticketKey(ticketID)

	var userID uuid.UUID
	err := s.q.Get(ctx, k, &userID)
	if cache.IsNotFoundError(err) {
		return uuid.Nil, cache.ErrNotFound
	}
	if err != nil {
		return uuid.Nil, cache.NewError(err, cache.ScopeTicket)
	}

	_ = s.q.Delete(ctx, k)

	return userID, nil
}
