package channel

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/repository"
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type Service struct {
	store repository.Store
}

func NewService(store repository.Store) *Service {
	return &Service{store: store}
}

type CreateDMParams struct {
	ActorID uuid.UUID
	PeerID  uuid.UUID
}

func (s *Service) FindOrCreateDM(ctx context.Context, p CreateDMParams) (View, error) {
	if p.ActorID == p.PeerID {
		return View{}, apperr.New(apperr.CodeBadRequest, "cannot create a DM channel with yourself")
	}

	// 1. Check for existing symmetric relationship
	channelID, err := s.store.FindSharedDMChannel(ctx, repository.FindSharedDMChannelParams{
		UserID:   pgtype.UUID{Bytes: p.ActorID, Valid: true},
		UserID_2: pgtype.UUID{Bytes: p.PeerID, Valid: true},
	})

	if err == nil {
		// channelID is already a pgtype.UUID, so pass it directly.
		// If your SQLC generated method expects pgtype.UUID, this works:
		row, err := s.store.ChannelGetByID(ctx, channelID)
		if err != nil {
			return View{}, apperr.NewDBError(err)
		}
		return NewView(row), nil
	}

	if !errors.Is(err, pgx.ErrNoRows) {
		return View{}, apperr.NewDBError(err)
	}

	// 2. Atomic Provisioning
	var newChannel repository.Channel
	err = s.store.ExecTx(ctx, func(qtx *repository.Queries) error {
		var txErr error

		// Use qtx here, not ctx
		newChannel, txErr = qtx.CreateChannel(ctx, repository.CreateChannelParams{
			Type: int16(TypeDM),
		})
		if txErr != nil {
			return txErr
		}

		// Use qtx here, not ctx
		if txErr = qtx.AddChannelMember(ctx, repository.AddChannelMemberParams{
			ChannelID: newChannel.ID,
			UserID:    pgtype.UUID{Bytes: p.ActorID, Valid: true},
		}); txErr != nil {
			return txErr
		}

		return qtx.AddChannelMember(ctx, repository.AddChannelMemberParams{
			ChannelID: newChannel.ID,
			UserID:    pgtype.UUID{Bytes: p.PeerID, Valid: true},
		})
	})

	if err != nil {
		return View{}, apperr.NewDBError(err)
	}

	return NewView(newChannel), nil
}
