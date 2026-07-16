package me

import (
	"bonfire-api/internal/user"
	"context"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

type Service struct {
	user *user.Service
}

func NewService(

	user *user.Service,
) *Service {
	return &Service{
		user: user,
	}
}

func (s *Service) GetByID(ctx context.Context, userID uuid.UUID) (View, error) {
	var (
		u  user.User
		up user.UserProfile
	)

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		u, err = s.user.GetByID(gCtx, userID)
		return err
	})

	g.Go(func() error {
		var err error
		up, err = s.user.GetProfileByUserID(gCtx, userID)
		return err
	})

	if err := g.Wait(); err != nil {
		return View{}, err
	}

	return ToView(u, up), nil
}
