package gateway

import (
	"net/http"

	"bonfire-api/internal/fields"
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/presence"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Handler struct {
	hub         *Hub
	userCache   UserCache
	ticketCache TicketCache
	bind        *httpio.Bind
}

func NewHandler(hub *Hub, userCache UserCache, ticketCache TicketCache, bind *httpio.Bind) *Handler {
	return &Handler{
		hub:         hub,
		userCache:   userCache,
		ticketCache: ticketCache,
		bind:        bind,
	}
}

type ServeWSQuery struct {
	TicketID uuid.UUID `form:"ticketId" validate:"required,uuid"`
	Presence string    `form:"presence" mod:"text" validate:"required,max=12"`
}

func (h *Handler) ServeWS(w http.ResponseWriter, r *http.Request) error {
	var query ServeWSQuery
	if err := h.bind.Query(r, &query); err != nil {
		return err
	}

	ticketID, err := fields.ParseID(query.TicketID)
	if err != nil {
		return err
	}

	ctx := r.Context()

	userID, sessionID, err := h.ticketCache.Punch(ctx, ticketID)
	if err != nil {
		return err
	}

	userPresence, err := presence.ParseString(query.Presence)
	if err != nil || !userPresence.IsValid() {
		userPresence = presence.NewOnline()
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return err
	}

	client := NewClient(ctx, userID, sessionID, conn)
	h.hub.Register(client, userPresence)
	client.StartPumps(h.hub)

	return nil
}
