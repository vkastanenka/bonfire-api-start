package handler

import (
	"context"
	"net/http"

	"bonfire-api/internal/httpio"
	"bonfire-api/internal/user"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

type MeUserService interface {
	Get(ctx context.Context, id uuid.UUID) (*user.User, error)
}

type MeHandler struct {
	users MeUserService
	// relationships relationship.Service
	// channels      channel.Service
	// presence      presence.Service
}

func NewMeHandler(
	u MeUserService,
	// r relationship.Service,
	// c channel.Service,
	// p presence.Service,
) *MeHandler {
	return &MeHandler{
		users: u,
		// relationships: r,
		// channels:      c,
		// presence:      p,
	}
}

type MeResponse struct {
	User UserMeResponse `json:"user"`
	// Relationships []relationship.Perspective             `json:"relationships"`
	// Channels      []channel.PrivateChannelResponse       `json:"channels"`
	// Presences     map[uuid.UUID]presence.Presence        `json:"presences"`
}

func (h *MeHandler) Get(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	userID, err := httpio.CtxGetUserID(ctx)
	if err != nil {
		return err
	}

	g, gCtx := errgroup.WithContext(ctx)

	var (
		meUser *user.User
		// relPersp []relationship.Perspective
		// privChannels []channel.PrivateChannel
	)

	// 1. Fetch Current User
	g.Go(func() error {
		var err error
		meUser, err = h.users.Get(gCtx, userID)
		return err
	})

	// 2. Fetch Relationships / Friends List
	// g.Go(func() error {
	// 	var err error
	// 	relPersp, err = h.relationships.ListPerspectives(gCtx, userID, nil)
	// 	return err
	// })

	// 3. Fetch User's Private DM Channels
	// g.Go(func() error {
	// 	var err error
	// 	privChannels, err = h.channels.ListPrivateForUser(gCtx, userID)
	// 	return err
	// })

	// Wait for all concurrent queries to complete
	if err := g.Wait(); err != nil {
		return err
	}

	// 4. Extract Peer IDs + Self ID and fetch real-time presences
	// peerIDs := append(extractPeerIDs(relPersp), userID)
	// presences, err := h.presence.GetBulk(ctx, peerIDs)
	// if err != nil {
	// 	return err
	// }

	// 5. Construct mapped DTO response
	httpio.RespondOK(w, r, MeResponse{
		User: ToUserMeResponse(*meUser),
		// Relationships: ToRelationshipResponse(relPersp),
		// Channels:      ToChannelResponses(privChannels),
		// Presences:     presences,
	})
	return nil
}
