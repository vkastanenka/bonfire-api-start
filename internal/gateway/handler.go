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
	h.hub.Register(client)
	client.StartPumps(h.hub)

	return nil
}

// func HandlePresenceUpdate(
// 	userCache *cache.UserCache,
// 	broadcaster *Broadcaster,
// 	relationshipService RelationshipService, // Interface to fetch recipient IDs
// ) MessageHandler {
// 	return func(ctx context.Context, client *Client, data json.RawMessage) error {
// 		var req ClientPresenceUpdatePayload
// 		if err := json.Unmarshal(data, &req); err != nil {
// 			return err
// 		}

// 		// 1. Parse and validate using your domain type
// 		p, err := user.ParsePresenceString(req.Presence)
// 		if err != nil || !p.IsValid() || p.IsOffline() {
// 			return user.ErrPresenceInvalid()
// 		}

// 		userID, err := fields.ParseID(client.UserID)
// 		if err != nil {
// 			return err
// 		}

// 		// 2. Persist the updated presence state in Redis
// 		if err := userCache.SetPresence(ctx, userID, p); err != nil {
// 			return err
// 		}

// 		// 3. Resolve target users (e.g., friends, mutual server members)
// 		recipientIDs, err := relationshipService.GetPresenceSubscribers(ctx, userID)
// 		if err != nil {
// 			return err
// 		}

// 		// 4. Delegate fan-out broadcasting
// 		eventPayload := PresenceChangeEventPayload{
// 			UserID:   userID,
// 			Presence: p,
// 		}

// 		return broadcaster.PublishToUsers(
// 			ctx,
// 			recipientIDs,
// 			EventTypePresenceChange,
// 			eventPayload,
// 		)
// 	}
// }
