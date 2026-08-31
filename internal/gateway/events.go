package gateway

// func NewUpdatePresenceHandler(events *user.Events) MessageHandler {
// 	return func(ctx context.Context, client *Client, data json.RawMessage) error {
// 		var payload user.WSUpdatePresenceData
// 		if err := json.Unmarshal(data, &payload); err != nil {
// 			return fmt.Errorf("invalid presence payload format: %w", err)
// 		}

// 		// Delegate to the domain service, passing gateway-specific context/client fields
// 		return events.UpdatePresence(ctx, client.UserID, payload)
// 	}
// }

// hub.RegisterMessageHandler(WSMsgHeartbeat, func(ctx context.Context, client *Client, data json.RawMessage) error {
//     var payload HeartbeatPayload
//     if err := json.Unmarshal(data, &payload); err != nil {
//         return err
//     }

//     // Delegate directly to the user service
//     return h.userService.HandleHeartbeat(ctx, client.UserID, h.id.String(), payload.Presence)
// })
