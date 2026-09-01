package gateway

import (
	"bonfire-api/internal/fields"
	"bonfire-api/internal/user"
	"context"
	"encoding/json"
	"log/slog"
	"time"
)

func NewHeartbeatHandler(userService UserService, nodeID fields.ID) MessageHandler {
	return func(ctx context.Context, client *Client, data json.RawMessage) error {
		var payload struct {
			Presence *int `json:"presence,omitempty"`
		}

		if len(data) > 0 {
			if err := json.Unmarshal(data, &payload); err != nil {
				return err
			}
		}

		var presence user.Presence
		if payload.Presence != nil {
			p, err := user.ParsePresence(*payload.Presence)
			if err != nil {
				return err
			}
			presence = p
		}

		reqCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
		defer cancel()

		if err := userService.HandleHeartbeat(reqCtx, client.UserID, nodeID, presence); err != nil {
			slog.ErrorContext(ctx, "failed to handle heartbeat", "user_id", client.UserID, "error", err)
			return err
		}

		return nil
	}
}
