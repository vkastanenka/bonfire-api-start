package gateway

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/cache"
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/presence.go"
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
	Ticket   uuid.UUID         `form:"ticket" validate:"required"`
	Presence presence.Presence `form:"presence" validate:"omitempty,presence"`
}

func (h *Handler) ServeWS(w http.ResponseWriter, r *http.Request) error {
	query, err := httpio.BindQuery[ServeWSQuery](r)
	if err != nil {
		return err
	}

	ctx := r.Context()
	ticketKey := cache.WSTicketKey(query.Ticket)

	var userIDStr string
	if err := h.cache.Get(ctx, ticketKey, &userIDStr); err != nil {
		return apperr.NewUnauthorized(err, "Websocket connection ticket is invalid or expired.")
	}

	_ = h.cache.Delete(ctx, ticketKey)

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return apperr.NewInternal(err, "Failed reconstructing identity signatures from token.")
	}

	if query.Presence == presence.PresenceUnknown {
		query.Presence = presence.PresenceOnline
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return err
	}

	client := &Client{
		UserID: userID,
		// Presence: query.Presence,
		Conn: conn,
		Send: make(chan []byte, 256),
	}

	h.hub.register <- client

	go client.writePump()
	go client.readPump(h.hub)

	return nil
}
