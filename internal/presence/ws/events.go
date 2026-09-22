package presencews

import (
	"context"

	"bonfire-api/internal/gateway"
	"bonfire-api/internal/presence"

	"github.com/google/uuid"
)

// HeartbeatService abstracts the heartbeat functionality required by this WS adapter.
type HeartbeatService interface {
	HandleHeartbeat(ctx context.Context, userID, nodeID, sessionID uuid.UUID, newPresence presence.Presence) error
}

type HeartbeatPayload struct {
	Presence *presence.Presence `json:"presence,omitempty"`
}

// Register attaches presence WebSocket handlers to the central Hub.
func Register(hub *gateway.Hub, service HeartbeatService) {
	hub.RegisterHandler("presence:heartbeat", newHeartbeatHandler(service))
}

func newHeartbeatHandler(service HeartbeatService) gateway.MessageHandler {
	return gateway.BindHandler(func(ctx context.Context, client gateway.ClientContext, payload HeartbeatPayload) error {
		var newPresence presence.Presence
		if payload.Presence != nil {
			newPresence = *payload.Presence
		}

		return service.HandleHeartbeat(
			ctx,
			client.UserID,
			client.NodeID,
			client.SessionID,
			newPresence,
		)
	})
}
