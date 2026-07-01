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

// PostMessage adds security checks before persisting
func (s *Service) PostMessage(ctx context.Context, userID uuid.UUID, p SendReq) (View, error) {
	// 1. Security Check: Verify user is a channel member
	// This ensures a user cannot force-post into a channel they don't belong to
	isMember, err := s.store.IsUserInChannel(ctx, repository.IsUserInChannelParams{
		ChannelID: pgtype.UUID{Bytes: p.ChannelID, Valid: true},
		UserID:    pgtype.UUID{Bytes: userID, Valid: true},
	})
	if err != nil || !isMember {
		return View{}, apperr.New(apperr.CodeForbidden, "you are not a member of this channel")
	}

	// 2. Persist
	row, err := s.store.InsertMessage(ctx, repository.InsertMessageParams{
		ChannelID: pgtype.UUID{Bytes: p.ChannelID, Valid: true},
		UserID:    pgtype.UUID{Bytes: userID, Valid: true},
		Content:   p.Content,
	})
	if err != nil {
		return View{}, apperr.NewDBError(err)
	}

	return NewView(row), nil
}

// GetMessages handles pagination for initial channel load
func (s *Service) GetMessages(ctx context.Context, channelID uuid.UUID, limit, offset int32) ([]View, error) {
	rows, err := s.store.MessageListByChannel(ctx, repository.MessageListByChannelParams{
		ChannelID: pgtype.UUID{Bytes: channelID, Valid: true},
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		return nil, apperr.NewDBError(err)
	}

	views := make([]View, len(rows))
	for i, row := range rows {
		views[i] = NewView(row)
	}
	return views, nil
}
