package gateway

import (
	"context"
	"log/slog"
	"net/http"

	"bonfire-api/internal/fields"
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/user"

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

	userPresence, err := user.ParsePresenceString(query.Presence)
	if err != nil || !userPresence.IsValid() {
		userPresence = user.NewPresenceOnline()
	}

	if err := h.userCache.SetPresence(context.WithoutCancel(ctx), userID, userPresence); err != nil {
		slog.ErrorContext(ctx, "Failed to set initial user presence on websocket connect", "user_id", userID, "error", err)
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return err
	}

	client := NewClient(ctx, userID.UUID(), sessionID.UUID(), conn)
	h.hub.register <- client

	client.StartPumps(h.hub)

	return nil
}
