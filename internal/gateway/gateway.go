package gateway

import (
	"context"
	"net/http"

	"bonfire-api/internal/errs"
	"bonfire-api/internal/httpio"

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
	bind  *httpio.Bind
}

func NewHandler(hub *Hub, cache TicketCacher, bind *httpio.Bind) *Handler {
	return &Handler{
		hub:   hub,
		cache: cache,
		bind:  bind,
	}
}

type ServeWSQuery struct {
	TicketID string  `form:"ticket-id" validate:"required,uuid"`
	Presence *string `form:"presence"  validate:"omitempty,presence"`
}

func (h *Handler) ServeWS(w http.ResponseWriter, r *http.Request) error {
	var query ServeWSQuery
	err := h.bind.Query(r, &query)
	if err != nil {
		return err
	}

	_, err = uuid.Parse(query.TicketID)
	if err != nil {
		return errs.InvalidArgument("Invalid ticket format.").
			Meta("ticket-id", "Must be a valid UUID v4 format.").
			FieldViolation("ticket-id", "Must be a valid UUID v4 format.", "INVALID_UUID").
			Wrap(err)
	}

	ctx := r.Context()
	// ticketKey := cache.WSTicketKey(ticketId)
	ticketKey := ""
	var ticket uuid.UUID

	err = h.cache.Get(ctx, ticketKey, &ticket)
	if err != nil {
		return errs.Unauthenticated("Websocket connection ticket is invalid or expired.").Wrap(err)
	}

	_ = h.cache.Delete(ctx, ticketKey)

	// userPresence := user.PresenceOnline
	// if query.Presence != nil {
	// 	userPresence = user.ParsePresence(*query.Presence)
	// }

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return err
	}

	client := &Client{
		UserID:   ticket,
		Presence: nil,
		Conn:     conn,
		Send:     make(chan []byte, 256),
	}
	h.hub.register <- client

	go client.writePump()
	go client.readPump(h.hub)

	return nil
}
