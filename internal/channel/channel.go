package channel

import (
	"time"

	"bonfire-api/internal/repository"

	"github.com/google/uuid"
)

type Type int16

const (
	TypeGuildText     Type = 0
	TypeDM            Type = 1
	TypeGuildVoice    Type = 2
	TypeGroupDM       Type = 3
	TypeGuildCategory Type = 4
)

func (t Type) String() string {
	switch t {
	case TypeGuildText:
		return "guild_text"
	case TypeDM:
		return "dm"
	case TypeGuildVoice:
		return "guild_voice"
	case TypeGroupDM:
		return "group_dm"
	case TypeGuildCategory:
		return "guild_category"
	default:
		return "unknown"
	}
}

// View matches your clean REST formatting metrics.
type View struct {
	ID        uuid.UUID  `json:"id"`
	Type      string     `json:"type"`
	GuildID   *uuid.UUID `json:"guild_id,omitempty"`
	Name      string     `json:"name,omitempty"` // Contextually hydrated if Type == TypeDM
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

func NewView(row repository.Channel) View {
	var gID *uuid.UUID
	if row.GuildID.Valid {
		id := uuid.UUID(row.GuildID.Bytes)
		gID = &id
	}

	return View{
		ID:        row.ID.Bytes,
		Type:      Type(row.Type).String(),
		GuildID:   gID,
		Name:      row.Name.String,
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}
}
