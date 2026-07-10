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

	ticket, err := h.service.WSTicket(r.Context(), WSTicketData{
		UserID: claims.UserID,
	})
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, WSTicketResponse{Ticket: ticket.String()})
	return nil
}

type WSTicketData struct {
	UserID uuid.UUID `json:"user_id"`
}

func (s *Service) WSTicket(ctx context.Context, p WSTicketData) (uuid.UUID, error) {
	ticketID, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, apperr.NewInternal(err, "Failed to generate websocket ticket")
	}

	ticketKey := cache.WSTicketKey(ticketID)
	err = s.cache.Set(ctx, ticketKey, p, WSTicketTTL)
	if err != nil {
		return uuid.Nil, err
	}

	return ticketID, nil
}
