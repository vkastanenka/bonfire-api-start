package gateway

import (
	"bonfire-api/internal/user"
	"context"
	"encoding/json"
	"fmt"
)

func NewUpdatePresenceHandler(events *user.Events) MessageHandler {
	return func(ctx context.Context, client *Client, data json.RawMessage) error {
		var payload user.WSUpdatePresenceData
		if err := json.Unmarshal(data, &payload); err != nil {
			return fmt.Errorf("invalid presence payload format: %w", err)
		}

		// Delegate to the domain service, passing gateway-specific context/client fields
		return events.UpdatePresence(ctx, client.UserID, payload)
	}
}
