package handler

import (
	"net/http"

	"bonfire-api/internal/httpio"

	"github.com/google/uuid"
)

type RelationHandler struct {
	service RelationService
	bind    *httpio.Bind
}

func NewRelationHandler(service RelationService, bind *httpio.Bind) *RelationHandler {
	return &RelationHandler{
		service: service,
		bind:    bind,
	}
}

type RelationPeerPath struct {
	PeerID uuid.UUID `path:"peerId" validate:"required,uuid"`
}

func (h *RelationHandler) GetPeer(w http.ResponseWriter, r *http.Request) error {
	actorID, err := httpio.CtxGetUserID(r.Context())
	if err != nil {
		return err
	}

	var path RelationPeerPath
	if err := h.bind.Path(r, &path); err != nil {
		return err
	}

	peer, err := h.service.GetPeer(r.Context(), actorID.UUID(), path.PeerID)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, peer)
	return nil
}

// func (h *RelationHandler) GetFriends(w http.ResponseWriter, r *http.Request) error {
// 	actorID, err := httpio.CtxGetUserID(r.Context())
// 	if err != nil {
// 		return err
// 	}

// 	peers, err := h.service.GetPeers(r.Context(), actorID.UUID(), relation.NewTypeFriends().String())
// 	if err != nil {
// 		return err
// 	}

// 	httpio.RespondOK(w, r, peers)
// 	return nil
// }

// func (h *RelationHandler) GetPending(w http.ResponseWriter, r *http.Request) error {
// 	actorID, err := httpio.CtxGetUserID(r.Context())
// 	if err != nil {
// 		return err
// 	}

// 	peers, err := h.service.GetPeers(r.Context(), actorID.UUID(), relation.NewTypePending().String())
// 	if err != nil {
// 		return err
// 	}

// 	httpio.RespondOK(w, r, peers)
// 	return nil
// }

// func (h *RelationHandler) GetBlocked(w http.ResponseWriter, r *http.Request) error {
// 	actorID, err := httpio.CtxGetUserID(r.Context())
// 	if err != nil {
// 		return err
// 	}

// 	peers, err := h.service.GetPeers(r.Context(), actorID.UUID(), relation.NewTypeBlocked().String())
// 	if err != nil {
// 		return err
// 	}

// 	httpio.RespondOK(w, r, peers)
// 	return nil
// }

func (h *RelationHandler) SendRequest(w http.ResponseWriter, r *http.Request) error {
	actorID, err := httpio.CtxGetUserID(r.Context())
	if err != nil {
		return err
	}

	var path RelationPeerPath
	if err := h.bind.Path(r, &path); err != nil {
		return err
	}

	if err := h.service.TransitionPending(r.Context(), actorID.UUID(), path.PeerID); err != nil {
		return err
	}

	httpio.RespondNoContent(w)
	return nil
}

func (h *RelationHandler) AcceptRequest(w http.ResponseWriter, r *http.Request) error {
	actorID, err := httpio.CtxGetUserID(r.Context())
	if err != nil {
		return err
	}

	var path RelationPeerPath
	if err := h.bind.Path(r, &path); err != nil {
		return err
	}

	if err := h.service.TransitionFriends(r.Context(), actorID.UUID(), path.PeerID); err != nil {
		return err
	}

	httpio.RespondNoContent(w)
	return nil
}

func (h *RelationHandler) BlockUser(w http.ResponseWriter, r *http.Request) error {
	actorID, err := httpio.CtxGetUserID(r.Context())
	if err != nil {
		return err
	}

	var path RelationPeerPath
	if err := h.bind.Path(r, &path); err != nil {
		return err
	}

	if err := h.service.TransitionBlocked(r.Context(), actorID.UUID(), path.PeerID); err != nil {
		return err
	}

	httpio.RespondNoContent(w)
	return nil
}

func (h *RelationHandler) RemoveRelation(w http.ResponseWriter, r *http.Request) error {
	actorID, err := httpio.CtxGetUserID(r.Context())
	if err != nil {
		return err
	}

	var path RelationPeerPath
	if err := h.bind.Path(r, &path); err != nil {
		return err
	}

	if err := h.service.DeleteByUserID(r.Context(), actorID.UUID(), path.PeerID); err != nil {
		return err
	}

	httpio.RespondNoContent(w)
	return nil
}
