package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// ClientContext contains metadata about the client connection sending the frame.
type ClientContext struct {
	UserID    uuid.UUID
	SessionID uuid.UUID
	NodeID    uuid.UUID
}

// MessageHandler is the raw frame handler stored in Hub.handlers.
type MessageHandler func(ctx context.Context, client *Client, data json.RawMessage) error

// TypedMessageHandler processes strongly typed gateway payloads with client context.
type TypedMessageHandler[T any] func(ctx context.Context, client ClientContext, payload T) error

// BindHandler adapts a TypedMessageHandler into a standard gateway MessageHandler,
// handling JSON payload unmarshaling and extracting ClientContext.
func BindHandler[T any](fn TypedMessageHandler[T]) MessageHandler {
	return func(ctx context.Context, client *Client, data json.RawMessage) error {
		var payload T
		if len(data) > 0 && !bytes.Equal(data, []byte("null")) {
			if err := json.Unmarshal(data, &payload); err != nil {
				return fmt.Errorf("failed to unmarshal gateway message: %w", err)
			}
		}

		clientCtx := ClientContext{
			UserID:    client.UserID,
			SessionID: client.SessionID,
			NodeID:    client.NodeID,
		}

		return fn(ctx, clientCtx, payload)
	}
}
