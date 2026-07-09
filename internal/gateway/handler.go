package gateway

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/cache"
	"bonfire-api/internal/httpio"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Handler struct {
	hub *Hub
}

func NewHandler(hub *Hub) *Handler {
	return &Handler{hub: hub}
}

func (h *Handler) ServeWS(w http.ResponseWriter, r *http.Request) error {
	userID, err := httpio.GetCtxUserID(r.Context())
	if err != nil {
		return apperr.NewUnauthorized(err, "")
	}

	statusParam := r.URL.Query().Get("status")
	if statusParam == "" {
		statusParam = "online"
	}
	initialPresence := cache.ParsePresence(statusParam)

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return err
	}

	client := &Client{
		UserID:   userID,
		Presence: initialPresence,
		Conn:     conn,
		Send:     make(chan []byte, 256),
	}

	h.hub.register <- client

	go client.writePump()
	go client.readPump(h.hub)

	return nil
}
