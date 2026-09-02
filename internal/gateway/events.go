package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"bonfire-api/internal/fields"
	"bonfire-api/internal/presence"
)

type MessageHandler func(ctx context.Context, client *Client, data json.RawMessage) error

type HeartbeatPayload struct {
	Presence *int `json:"presence,omitempty"`
}

// ParsePayload parses optional raw JSON payloads safely.
func ParsePayload[T any](op string, rawMessage json.RawMessage) (*T, error) {
	if len(rawMessage) == 0 || bytes.Equal(rawMessage, []byte("null")) {
		var empty T
		return &empty, nil
	}

	var msg T
	if err := json.Unmarshal(rawMessage, &msg); err != nil {
		return nil, fmt.Errorf("failed to parse %s gateway payload: %w", op, err)
	}
	return &msg, nil
}

// NewHeartbeatHandler handles websocket heartbeats and optional presence updates.
func NewHeartbeatHandler(service *Service, nodeID fields.ID) MessageHandler {
	return func(ctx context.Context, client *Client, data json.RawMessage) error {
		payload, err := ParsePayload[HeartbeatPayload]("heartbeat", data)
		if err != nil {
			return err
		}

		var newPresence presence.Presence
		if payload.Presence != nil {
			p, err := presence.Parse(*payload.Presence)
			if err != nil {
				return fmt.Errorf("invalid presence value in heartbeat payload: %w", err)
			}
			newPresence = p
		}

		reqCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
		defer cancel()

		if err := service.HandleHeartbeat(reqCtx, client.UserID, nodeID, newPresence); err != nil {
			slog.ErrorContext(reqCtx, "failed to handle heartbeat", "user_id", client.UserID, "error", err)
			return err
		}

		return nil
	}
}
