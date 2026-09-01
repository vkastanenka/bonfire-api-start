package gateway

import (
	"bonfire-api/internal/fields"
	"bonfire-api/internal/user"
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

type GatewayCache interface{}

type UserCache interface {
	AddNode(ctx context.Context, userID fields.ID, nodeID string) error
	ClearNodes(ctx context.Context, userID fields.ID) error
	GetBatchPresence(ctx context.Context, userIDs []fields.ID) (map[fields.ID]user.Presence, error)
	GetNodes(ctx context.Context, userID fields.ID) ([]string, error)
	GetPresence(ctx context.Context, userID fields.ID) (user.Presence, error)
	RemoveNode(ctx context.Context, userID fields.ID, nodeID string) error
	RemoveNodeBatch(ctx context.Context, userIDs []fields.ID, nodeID string) error
	SetBatchPresence(ctx context.Context, items map[fields.ID]user.Presence) error
	SetPresence(ctx context.Context, userID fields.ID, p user.Presence) error
}

type UserService interface {
	AnonymizeBatch(ctx context.Context) error
	Disable(ctx context.Context, p user.DisableParams) error
	Get(ctx context.Context, userID uuid.UUID) (*user.User, error)
	GetBatch(ctx context.Context, ids []fields.ID) (map[fields.ID]*user.User, error)
	GetBatchPresence(ctx context.Context, userIDs []fields.ID) (map[fields.ID]user.Presence, error)
	HandleHeartbeat(ctx context.Context, userID fields.ID, nodeID fields.ID, newPresence user.Presence) error
	Publish(ctx context.Context, nodeIDs []fields.ID, userIDs []fields.ID, eventType string, payload json.RawMessage) error
	RegisterWSConnection(ctx context.Context, userID fields.ID, nodeID fields.ID, presence user.Presence) error
	RemoveBatchNode(ctx context.Context, userIDs []fields.ID, nodeID fields.ID) error
	ScheduleDelete(ctx context.Context, p user.ScheduleDeleteParams) error
	UnregisterWSConnection(ctx context.Context, userID fields.ID, nodeID fields.ID) error
	UpdateEmail(ctx context.Context, p user.UpdateEmailParams) (*user.User, error)
	UpdatePassword(ctx context.Context, p user.UpdatePasswordParams) error
	UpdatePreferredPresence(ctx context.Context, p user.UpdatePreferredPresenceParams) (*user.User, error)
	UpdateProfile(ctx context.Context, p user.UpdateProfileParams) (*user.User, error)
	UpdateUsername(ctx context.Context, p user.UpdateUsernameParams) (*user.User, error)
}

type TicketCache interface {
	Print(ctx context.Context, ticketID fields.ID, userID fields.ID, sessionID fields.ID) error
	Punch(ctx context.Context, ticketID fields.ID) (fields.ID, fields.ID, error)
}
