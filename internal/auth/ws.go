package auth

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/cache"
	"bonfire-api/internal/httpio"
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type CreateWSTicketResponse struct {
	Ticket string `json:"ticket"`
}

func (h *Handler) CreateWSTicket(w http.ResponseWriter, r *http.Request) error {
	userID, err := httpio.GetCtxUserID(r.Context())
	if err != nil {
		return err
	}

	ticket, err := h.service.CreateWSTicket(r.Context(), userID)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, CreateWSTicketResponse{Ticket: ticket.String()})
	return nil
}

func (s *Service) CreateWSTicket(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	ticket, err := uuid.NewV7()
	if err != nil {
		return uuid.UUID{}, apperr.NewInternal(err, "")
	}

	ticketKey := cache.WSTicketKey(ticket)
	err = s.cache.Set(ctx, ticketKey, userID.String(), 20*time.Second)
	if err != nil {
		return uuid.UUID{}, err
	}

	return ticket, nil
}
