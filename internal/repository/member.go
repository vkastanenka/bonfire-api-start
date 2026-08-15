package repository

import "bonfire-api/internal/db"

type MemberRepository struct {
	store db.Querier
}

func NewMemberRepository(store db.Querier) *MemberRepository {
	return &MemberRepository{store: store}
}
