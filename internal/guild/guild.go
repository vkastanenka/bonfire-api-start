package guild

import (
	"bonfire-api/internal/repository"
	"time"

	"github.com/google/uuid"
)

type View struct {
	ID          uuid.UUID `json:"id"`
	OwnerID     uuid.UUID `json:"owner_id"`
	CreatedAt   time.Time `json:"created_at"`
	Name        string    `json:"name"`
	IconURL     *string   `json:"icon_url"`
	BannerHex   string    `json:"banner_hex"`
	Description *string   `json:"description"`
	Visibility  int16     `json:"visibility"`
}

func NewView(row repository.GuildGetFullRow) View {
	return View{
		ID:          row.ID.Bytes,
		OwnerID:     row.OwnerID.Bytes,
		CreatedAt:   row.CreatedAt.Time,
		Name:        row.Name,
		IconURL:     &row.IconUrl.String, // Handle NullString logic based on your repo
		BannerHex:   row.BannerHex.String,
		Description: &row.Description.String,
		Visibility:  row.Visibility,
	}
}

type MemberView struct {
	ID       uuid.UUID `json:"id"`
	Username string    `json:"username"`
}

type RoleView struct {
	ID          uuid.UUID `json:"id"`
	GuildID     uuid.UUID `json:"guild_id"`
	Name        string    `json:"name"`
	ColorHex    string    `json:"color_hex"`
	Permissions int64     `json:"permissions"`
}
