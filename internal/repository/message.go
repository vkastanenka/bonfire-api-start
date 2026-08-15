package repository

import "bonfire-api/internal/db"

type MessageRepository struct {
	store db.Querier
}

func NewMessageRepository(store db.Querier) *MessageRepository {
	return &MessageRepository{store: store}
}
