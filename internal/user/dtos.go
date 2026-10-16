package user

import (
	"bonfire-api/internal/gateway"
	"time"

	"github.com/google/uuid"
)

type SummaryDTO struct {
	ID          uuid.UUID `json:"id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"displayName"`
	AvatarURL   *string   `json:"avatarUrl,omitempty"`
	IsDisabled  bool      `json:"isDisabled"`
}

func NewSummaryDTO(u *User) *SummaryDTO {
	if u == nil {
		return nil
	}
	return &SummaryDTO{
		ID:          u.ID,
		Username:    u.Username,
		DisplayName: u.DisplayName,
		AvatarURL:   u.AvatarURL,
		IsDisabled:  u.IsDisabled(),
	}
}

func NewSummaryDTOs(users []*User) []SummaryDTO {
	if len(users) == 0 {
		return nil
	}

	dtos := make([]SummaryDTO, 0, len(users))
	for _, u := range users {
		if dto := NewSummaryDTO(u); dto != nil {
			dtos = append(dtos, *dto)
		}
	}

	return dtos
}

func NewSummaryDTOMap(users map[uuid.UUID]*User) map[uuid.UUID]SummaryDTO {
	if len(users) == 0 {
		return make(map[uuid.UUID]SummaryDTO)
	}

	dtos := make(map[uuid.UUID]SummaryDTO, len(users))
	for id, u := range users {
		if dto := NewSummaryDTO(u); dto != nil {
			dtos[id] = *dto
		}
	}

	return dtos
}

type DTO struct {
	ID          uuid.UUID `json:"id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"displayName"`
	AvatarURL   *string   `json:"avatarUrl,omitempty"`
	Bio         *string   `json:"bio,omitempty"`
	BannerColor *string   `json:"bannerColor,omitempty"`
	IsDisabled  bool      `json:"isDisabled"`
	CreatedAt   time.Time `json:"createdAt"`
}

func NewDTO(u *User) *DTO {
	if u == nil {
		return nil
	}
	return &DTO{
		ID:          u.ID,
		Username:    u.Username,
		DisplayName: u.DisplayName,
		AvatarURL:   u.AvatarURL,
		Bio:         u.Bio,
		BannerColor: u.BannerColor,
		IsDisabled:  u.IsDisabled(),
		CreatedAt:   u.CreatedAt,
	}
}

func NewDTOs(users []*User) []DTO {
	if len(users) == 0 {
		return nil
	}

	dtos := make([]DTO, 0, len(users))
	for _, u := range users {
		if dto := NewDTO(u); dto != nil {
			dtos = append(dtos, *dto)
		}
	}

	return dtos
}

func NewDTOMap(users map[uuid.UUID]*User) map[uuid.UUID]DTO {
	if len(users) == 0 {
		return make(map[uuid.UUID]DTO)
	}

	dtos := make(map[uuid.UUID]DTO, len(users))
	for id, u := range users {
		if dto := NewDTO(u); dto != nil {
			dtos[id] = *dto
		}
	}

	return dtos
}

type MeDTO struct {
	ID                     uuid.UUID         `json:"id"`
	Email                  string            `json:"email"`
	Username               string            `json:"username"`
	DisplayName            string            `json:"displayName"`
	AvatarURL              *string           `json:"avatarUrl,omitempty"`
	Bio                    *string           `json:"bio,omitempty"`
	BannerColor            *string           `json:"bannerColor,omitempty"`
	IsVerified             bool              `json:"isVerified"`
	VerifiedAt             *time.Time        `json:"verifiedAt,omitempty"`
	PreferredPresence      *gateway.Presence `json:"preferredPresence,omitempty"`
	PreferredPresenceUntil *time.Time        `json:"preferredPresenceUntil,omitempty"`
	CreatedAt              time.Time         `json:"createdAt"`
	UpdatedAt              time.Time         `json:"updatedAt"`
}

func NewMeDTO(u *User) *MeDTO {
	if u == nil {
		return nil
	}
	return &MeDTO{
		ID:                     u.ID,
		Email:                  u.Email,
		Username:               u.Username,
		DisplayName:            u.DisplayName,
		AvatarURL:              u.AvatarURL,
		Bio:                    u.Bio,
		BannerColor:            u.BannerColor,
		IsVerified:             u.IsVerified(),
		VerifiedAt:             u.VerifiedAt,
		PreferredPresence:      u.PreferredPresence,
		PreferredPresenceUntil: u.PreferredPresenceUntil,
		CreatedAt:              u.CreatedAt,
		UpdatedAt:              u.UpdatedAt,
	}
}
