package gateway

import (
	"bonfire-api/internal/appctx"
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/pkg/errs"
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Handler manages WebSocket connection upgrades, query verification, and client lifecycle registration.
type Handler struct {
	hub     *Hub
	service *Service
	bind    *httpio.Bind
}

// NewHandler initializes and returns a new gateway Handler instance.
func NewHandler(hub *Hub, service *Service, bind *httpio.Bind) *Handler {
	return &Handler{
		hub:     hub,
		service: service,
		bind:    bind,
	}
}

type PrintTicketResponse struct {
	Ticket uuid.UUID `json:"ticket"`
}

func (h *Handler) PrintTicket(w http.ResponseWriter, r *http.Request) error {
	claims, err := appctx.GetClaims(r.Context())
	if err != nil {
		return err
	}

	ticket, err := h.service.PrintTicket(r.Context(), PrintTicketParams{
		UserID: claims.UserID,
	})
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, PrintTicketResponse{Ticket: ticket})
	return nil
}

// ServeWSQuery defines the required query parameters extracted during the WebSocket handshake.
type ServeWSQuery struct {
	Ticket   uuid.UUID `form:"ticket" validate:"required"`
	Presence Presence  `form:"presence" validate:"required"`
}

// ServeWS validates incoming query parameters, punches the connection ticket, upgrades the HTTP
// connection to a WebSocket, and registers the active client with the hub.
func (h *Handler) ServeWS(w http.ResponseWriter, r *http.Request) error {
	var query ServeWSQuery
	if err := h.bind.Query(r, &query); err != nil {
		return err
	}

	userID, sessionID, clientID, err := h.service.PunchTicket(r.Context(), query.Ticket)
	if err != nil {
		return err
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return errs.Internal("Websocket connection upgrade failed.").Wrap(err)
	}

	client := NewClient(context.Background(), clientID, userID, sessionID, h.hub, conn)
	h.hub.RegisterClient(client, query.Presence)
	client.StartPumps()

	return nil
}
