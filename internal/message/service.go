package message

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/repository"
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type Service struct {
	store repository.Store
}

func NewService(store repository.Store) *Service {
	return &Service{store: store}
}

type SendParams struct {
	UserID         uuid.UUID
	ConversationID uuid.UUID
	Content        string
}

func (s *Service) Send(ctx context.Context, p SendParams) (View, error) {
	// 1. Verify user is in the conversation (Access Control)
	isMember, err := s.store.IsUserInConversation(ctx, repository.IsUserInConversationParams{
		UserID:         pgtype.UUID{Bytes: p.UserID, Valid: true},
		ConversationID: pgtype.UUID{Bytes: p.ConversationID, Valid: true},
	})
	if err != nil || !isMember {
		return View{}, apperr.New(apperr.CodeForbidden, "not a member of this conversation")
	}

	// 2. Create message
	row, err := s.store.MessageCreate(ctx, repository.MessageCreateParams{
		ConversationID: pgtype.UUID{Bytes: p.ConversationID, Valid: true},
		AuthorID:       pgtype.UUID{Bytes: p.UserID, Valid: true},
		Content:        p.Content,
	})
	if err != nil {
		return View{}, apperr.NewDBError(err)
	}

	return NewView(row), nil
}

func (s *Service) List(ctx context.Context, conversationID uuid.UUID) ([]View, error) {
	rows, err := s.store.MessageListByConversation(ctx, pgtype.UUID{Bytes: conversationID, Valid: true})
	if err != nil {
		return nil, apperr.NewDBError(err)
	}

	views := make([]View, len(rows))
	for i, row := range rows {
		views[i] = NewView(row)
	}
	return views, nil
}
