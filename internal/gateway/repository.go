package gateway

import (
	"bonfire-api/internal/fields"
	"bonfire-api/internal/user"
	"context"
)

type GatewayCache interface{}

type UserCache interface {
	AddChannelID(ctx context.Context, userID fields.ID, channelID fields.ID) error
	AddFriendID(ctx context.Context, userID fields.ID, friendID fields.ID) error
	Delete(ctx context.Context, id fields.ID) error
	DeleteBatch(ctx context.Context, ids []fields.ID) error
	Get(ctx context.Context, id fields.ID) (*user.User, error)
	GetBatch(ctx context.Context, ids []fields.ID) (map[fields.ID]*user.User, []fields.ID, error)
	GetChannelIDs(ctx context.Context, userID fields.ID) ([]fields.ID, error)
	GetFriendIDs(ctx context.Context, userID fields.ID) ([]fields.ID, error)
	GetPeerIDs(ctx context.Context, userID fields.ID) ([]fields.ID, error)
	RemoveChannelID(ctx context.Context, userID fields.ID, channelID fields.ID) error
	RemoveFriendID(ctx context.Context, userID fields.ID, friendID fields.ID) error
	Set(ctx context.Context, usr *user.User) error
	SetBatch(ctx context.Context, users map[fields.ID]*user.User) error
	SetChannelIDs(ctx context.Context, userID fields.ID, channelIDs []fields.ID) error
	SetFriendIDs(ctx context.Context, userID fields.ID, friendIDs []fields.ID) error
}

type PresenceCache interface {
	GetBatchNodes(ctx context.Context, userIDs []fields.ID) (map[fields.ID][]fields.ID, error)
	GetBatchPresence(ctx context.Context, userIDs []fields.ID) (map[fields.ID]user.Presence, error)
	GetPresence(ctx context.Context, userID fields.ID) (user.Presence, error)
	Heartbeat(ctx context.Context, userID fields.ID, nodeID fields.ID) error
	RegisterNode(ctx context.Context, userID fields.ID, nodeID fields.ID, presence user.Presence) (bool, user.Presence, error)
	RemoveBatchNodes(ctx context.Context, userIDs []fields.ID, nodeID fields.ID) error
	SetPresence(ctx context.Context, userID fields.ID, p user.Presence) error
	UnregisterNode(ctx context.Context, userID fields.ID, nodeID fields.ID) (bool, error)
}

type TicketCache interface {
	Print(ctx context.Context, ticketID fields.ID, userID fields.ID, sessionID fields.ID) error
	Punch(ctx context.Context, ticketID fields.ID) (fields.ID, fields.ID, error)
}

type OutboxRepository interface {
	Publish(ctx context.Context, eventType string, payload any, now fields.Timestamp) error
}

type TX interface {
	ExecTx(ctx context.Context, fn func(txCtx context.Context) error) error
}
