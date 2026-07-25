package handler

import (
	"context"
	"net/http"

	"bonfire-api/internal/httpio"
	"bonfire-api/internal/presence"
	"bonfire-api/internal/relationship"
	"bonfire-api/internal/user"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

type MeRelationshipService interface {
	ListPerspectives(ctx context.Context, userID uuid.UUID, filter *relationship.Variant) ([]relationship.Perspective, error)
	GetPerspective(ctx context.Context, userID, peerID uuid.UUID) (*relationship.Perspective, error)
	SendFriendRequest(ctx context.Context, actorID, targetID uuid.UUID) error
	AcceptFriendRequest(ctx context.Context, actorID, peerID uuid.UUID) error
	Block(ctx context.Context, actorID, peerID uuid.UUID) error
	DeleteVerified(ctx context.Context, actorID, peerID uuid.UUID) error
}

type MeUserService interface {
	Get(ctx context.Context, id uuid.UUID) (*user.User, error)
}

type MePresenceService interface {
	GetBulk(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]presence.Presence, error)
}

type Me struct {
	relationships MeRelationshipService
	users         MeUserService
	presence      MePresenceService
	// channels      channel.Service
	bind *httpio.Bind
}

func NewMe(
	r MeRelationshipService,
	u MeUserService,
	p MePresenceService,
	// c channel.Service,
	bind *httpio.Bind,
) *Me {
	return &Me{
		relationships: r,
		users:         u,
		presence:      p,
		// channels:      c,
		bind: bind,
	}
}

type MeGetResponse struct {
	Me            MeResponse                      `json:"me"`
	Relationships []RelationshipResponse          `json:"relationships"`
	Presences     map[uuid.UUID]presence.Presence `json:"presences"`
	// Channels      []channel.PrivateChannelResponse `json:"channels"`
}

func (h *Me) Get(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	userID, err := httpio.CtxGetUserID(ctx)
	if err != nil {
		return err
	}

	g, gCtx := errgroup.WithContext(ctx)

	var (
		meUser   *user.User
		relPersp []relationship.Perspective
		// privChannels []channel.PrivateChannel
	)

	g.Go(func() error {
		var err error
		meUser, err = h.users.Get(gCtx, userID)
		return err
	})

	g.Go(func() error {
		var err error
		relPersp, err = h.relationships.ListPerspectives(gCtx, userID, nil)
		return err
	})

	if err := g.Wait(); err != nil {
		return err
	}

	peerIDs := make([]uuid.UUID, 0, len(relPersp)+1)
	peerIDs = append(peerIDs, userID)
	for _, p := range relPersp {
		peerIDs = append(peerIDs, p.PeerID())
	}

	presences, err := h.presence.GetBulk(ctx, peerIDs)
	if err != nil {
		return err
	}

	relResponses := make([]RelationshipResponse, len(relPersp))
	for i, p := range relPersp {
		var peerPresence *presence.Presence
		if ps, ok := presences[p.PeerID()]; ok {
			peerPresence = &ps
		}
		relResponses[i] = ToRelationshipResponse(p, peerPresence)
	}

	httpio.RespondOK(w, r, MeGetResponse{
		Me:            ToMeResponse(*meUser),
		Relationships: relResponses,
		Presences:     presences,
	})
	return nil
}

type MeSendFriendRequestPath struct {
	ID uuid.UUID `path:"id" validate:"required,uuid"`
}

func (h *Me) SendFriendRequest(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	actorID, err := httpio.CtxGetUserID(ctx)
	if err != nil {
		return err
	}

	var path MeSendFriendRequestPath
	if err := h.bind.Path(r, &path); err != nil {
		return err
	}

	if err := h.relationships.SendFriendRequest(ctx, actorID, path.ID); err != nil {
		return err
	}

	h.respondWithPerspective(w, r, actorID, path.ID)
	return nil
}

type MeAcceptFriendRequestPath struct {
	ID uuid.UUID `path:"id" validate:"required,uuid"`
}

func (h *Me) AcceptFriendRequest(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	actorID, err := httpio.CtxGetUserID(ctx)
	if err != nil {
		return err
	}

	var path MeAcceptFriendRequestPath
	if err := h.bind.Path(r, &path); err != nil {
		return err
	}

	if err := h.relationships.AcceptFriendRequest(ctx, actorID, path.ID); err != nil {
		return err
	}

	h.respondWithPerspective(w, r, actorID, path.ID)
	return nil
}

type MeBlockPath struct {
	ID uuid.UUID `path:"id" validate:"required,uuid"`
}

func (h *Me) Block(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	actorID, err := httpio.CtxGetUserID(ctx)
	if err != nil {
		return err
	}

	var path MeBlockPath
	if err := h.bind.Path(r, &path); err != nil {
		return err
	}

	if err := h.relationships.Block(ctx, actorID, path.ID); err != nil {
		return err
	}

	h.respondWithPerspective(w, r, actorID, path.ID)
	return nil
}

type MeRemoveRelationshipPath struct {
	ID uuid.UUID `path:"id" validate:"required,uuid"`
}

func (h *Me) RemoveRelationship(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	actorID, err := httpio.CtxGetUserID(ctx)
	if err != nil {
		return err
	}

	var path MeRemoveRelationshipPath
	if err := h.bind.Path(r, &path); err != nil {
		return err
	}

	if err := h.relationships.DeleteVerified(ctx, actorID, path.ID); err != nil {
		return err
	}

	httpio.RespondNoContent(w)
	return nil
}

func (h *Me) respondWithPerspective(
	w http.ResponseWriter,
	r *http.Request,
	userID, peerID uuid.UUID,
) error {
	ctx := r.Context()

	perspective, err := h.relationships.GetPerspective(ctx, userID, peerID)
	if err != nil {
		return err
	}

	var peerPresence *presence.Presence
	presences, err := h.presence.GetBulk(ctx, []uuid.UUID{peerID})
	if err == nil {
		if ps, ok := presences[peerID]; ok {
			peerPresence = &ps
		}
	}

	httpio.RespondOK(w, r, ToRelationshipResponse(*perspective, peerPresence))
	return nil
}
