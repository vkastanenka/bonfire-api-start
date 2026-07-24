package store

import (
	"context"
	"fmt"
	"time"

	"bonfire-api/internal/cache"

	"github.com/google/uuid"
)

type TicketStore interface {
	Get(ctx context.Context, key string, dest interface{}) error
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key ...string) error
}

type Ticket struct {
	store TicketStore
}

func NewTicket(store TicketStore) *Ticket {
	return &Ticket{store: store}
}

func ticketKey(id uuid.UUID) string {
	return fmt.Sprintf("ticket:{%s}", id.String())
}

func (s *Ticket) SetTicket(ctx context.Context, ticketID, userID uuid.UUID, ttl time.Duration) error {
	k := ticketKey(ticketID)
	if err := s.store.Set(ctx, k, userID, ttl); err != nil {
		return cache.NewError(err, cache.ScopeTicket)
	}
	return nil
}

func (s *Ticket) ConsumeTicket(ctx context.Context, ticketID uuid.UUID) (uuid.UUID, error) {
	k := ticketKey(ticketID)

	var userID uuid.UUID
	err := s.store.Get(ctx, k, &userID)
	if cache.IsNotFoundError(err) {
		return uuid.Nil, cache.ErrNotFound
	}
	if err != nil {
		return uuid.Nil, cache.NewError(err, cache.ScopeTicket)
	}

	_ = s.store.Delete(ctx, k)

	return userID, nil
}
