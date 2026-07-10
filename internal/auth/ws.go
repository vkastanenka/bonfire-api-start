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

// TODO: Update to config
const WSTicketTTL = 20 * time.Second

type WSTicketResponse struct {
	Ticket string `json:"ticket"`
}

func (h *Handler) WSTicket(w http.ResponseWriter, r *http.Request) error {
	claims, err := httpio.GetCtxClaims(r.Context())
	if err != nil {
		return err
	}

	ticket, err := h.service.WSTicket(r.Context(), WSTicketParams{
		UserID:    claims.UserID,
		SessionID: claims.SessionID,
	})
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, WSTicketResponse{Ticket: ticket.String()})
	return nil
}

type WSTicketParams struct {
	UserID    uuid.UUID `json:"user_id"`
	SessionID uuid.UUID `json:"session_id"`
}

func (s *Service) WSTicket(ctx context.Context, p WSTicketParams) (uuid.UUID, error) {
	ticket, err := uuid.NewV7()
	if err != nil {
		return uuid.UUID{}, apperr.NewInternal(err, "")
	}

	ticketKey := cache.WSTicketKey(ticket)
	err = s.cache.Set(ctx, ticketKey, p, WSTicketTTL)
	if err != nil {
		return uuid.UUID{}, err
	}

	return ticket, nil
}
