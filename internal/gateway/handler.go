package gateway

import (
	"context"
	"net/http"

	"bonfire-api/internal/httpio"
	"bonfire-api/internal/pkg/errs"
	"bonfire-api/internal/presence"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Handler manages WebSocket connection upgrades, query verification, and client lifecycle registration.
type Handler struct {
	hub         *Hub
	ticketCache WSTicketCache
	bind        *httpio.Bind
}

// NewHandler initializes and returns a new gateway Handler instance.
func NewHandler(hub *Hub, ticketCache WSTicketCache, bind *httpio.Bind) *Handler {
	return &Handler{
		hub:         hub,
		ticketCache: ticketCache,
		bind:        bind,
	}
}

// ServeWSQuery defines the required query parameters extracted during the WebSocket handshake.
type ServeWSQuery struct {
	TicketID uuid.UUID         `form:"ticketId" validate:"required"`
	Presence presence.Presence `form:"presence" validate:"required"`
}

// ServeWS validates incoming query parameters, punches the connection ticket, upgrades the HTTP
// connection to a WebSocket, and registers the active client with the hub.
func (h *Handler) ServeWS(w http.ResponseWriter, r *http.Request) error {
	var query ServeWSQuery
	if err := h.bind.Query(r, &query); err != nil {
		return err
	}

	ctx := r.Context()

	userID, sessionID, err := h.ticketCache.Punch(ctx, query.TicketID)
	if err != nil {
		return err
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return errs.Internal("Websocket connection upgrade failed.").Wrap(err)
	}

	client := NewClient(context.Background(), h.hub.ID(), userID, sessionID, conn)
	h.hub.Register(client, query.Presence)
	client.StartPumps(h.hub)

	return nil
}
