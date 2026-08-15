package repository

import "bonfire-api/internal/db"

type ReactionRepository struct {
	store db.Querier
}

func NewReactionRepository(store db.Querier) *ReactionRepository {
	return &ReactionRepository{store: store}
}
