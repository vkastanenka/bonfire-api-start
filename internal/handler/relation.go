package handler

import (
	"net/http"

	"bonfire-api/internal/channel"
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/presence"
	"bonfire-api/internal/user"

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
	PeerID uuid.UUID `path:"peerId" validate:"required"`
}

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

type AcceptRequestResponsePayload struct {
	Channel      *channel.Channel
	ActorMember  *channel.Member
	PeerUser     user.View
	PeerPresence presence.Presence
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

	result, err := h.service.TransitionFriends(r.Context(), actorID.UUID(), path.PeerID)
	if err != nil {
		return err
	}

	payload := &AcceptRequestResponsePayload{
		Channel:      result.Channel,     // TODO: Update view
		ActorMember:  result.ActorMember, // TODO: Update view
		PeerUser:     user.ParseView(result.PeerUser),
		PeerPresence: result.PeerPresence,
	}

	httpio.RespondOK(w, r, payload)
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
