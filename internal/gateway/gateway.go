package gateway

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"bonfire-api/internal/httpio"
	"bonfire-api/internal/user"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type PresenceStore interface {
	SetPresence(ctx context.Context, userID uuid.UUID, p user.Presence) error
}

type TicketStore interface {
	SetTicket(ctx context.Context, ticketID, userID uuid.UUID, ttl time.Duration) error
	ConsumeTicket(ctx context.Context, ticketID uuid.UUID) (uuid.UUID, error)
}

type Handler struct {
	hub           *Hub
	presenceCache PresenceStore
	tickets       TicketStore
	bind          *httpio.Bind
}

func NewHandler(hub *Hub, presenceCache PresenceStore, tickets TicketStore, bind *httpio.Bind) *Handler {
	return &Handler{
		hub:           hub,
		presenceCache: presenceCache,
		tickets:       tickets,
		bind:          bind,
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

	ctx := r.Context()
	userID, err := h.tickets.ConsumeTicket(ctx, query.TicketID)
	if err != nil {
		return err
	}

	// Safely evaluate and resolve user presence
	userPresence, err := user.ParsePresenceString(query.Presence)
	if err != nil || !userPresence.IsValid() {
		userPresence = user.NewPresenceOnline()
	}

	// Set presence using a decoupled/safe context so it doesn't get aborted mid-flight
	if err := h.presenceCache.SetPresence(context.WithoutCancel(ctx), userID, userPresence); err != nil {
		slog.ErrorContext(ctx, "Failed to set initial user presence on websocket connect", "user_id", userID, "error", err)
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return err
	}

	client := &Client{
		UserID: userID,
		Conn:   conn,
		Send:   make(chan []byte, 256),
	}
	h.hub.register <- client

	go client.writePump()
	go client.readPump(h.hub)

	return nil
}
