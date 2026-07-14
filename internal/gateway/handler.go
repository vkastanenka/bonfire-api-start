package gateway

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/auth"
	"bonfire-api/internal/cache"
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/presence"
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type TicketCacher interface {
	Get(ctx context.Context, key string, dest interface{}) error
	Delete(ctx context.Context, key string) error
}

type Handler struct {
	hub   *Hub
	cache TicketCacher
}

func NewHandler(hub *Hub, cache TicketCacher) *Handler {
	return &Handler{
		hub:   hub,
		cache: cache,
	}
}

type ServeWSQuery struct {
	TicketID string  `form:"ticket_id" validate:"required"`
	Presence *string `form:"presence"  validate:"omitempty,presence"`
}

func (h *Handler) ServeWS(w http.ResponseWriter, r *http.Request) error {
	query, err := httpio.BindQuery[ServeWSQuery](r)
	if err != nil {
		return err
	}

	ticketId, err := uuid.Parse(query.TicketID)
	if err != nil {
		return apperr.NewInvalidInput(
			err,
			"Invalid ticket format.",
			apperr.Param("ticket_id", "Must be a valid UUID v4 format."),
		)
	}

	ctx := r.Context()
	ticketKey := cache.WSTicketKey(ticketId)
	var ticket auth.WSTicketData

	err = h.cache.Get(ctx, ticketKey, &ticket)
	if err != nil {
		return apperr.NewUnauthorized(err, "Websocket connection ticket is invalid or expired.")
	}

	_ = h.cache.Delete(ctx, ticketKey)

	userPresence := presence.PresenceOnline
	if query.Presence != nil {
		userPresence = presence.Parse(*query.Presence)
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return err
	}

	client := &Client{
		UserID:   ticket.UserID,
		Presence: userPresence,
		Conn:     conn,
		Send:     make(chan []byte, 256),
	}
	h.hub.register <- client

	go client.writePump()
	go client.readPump(h.hub)

	return nil
}
