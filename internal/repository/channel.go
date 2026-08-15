package repository

import (
	"bonfire-api/internal/db"
)

type ChannelRepository struct {
	store db.Querier
}

func NewChannelRepository(store db.Querier) *ChannelRepository {
	return &ChannelRepository{store: store}
}
