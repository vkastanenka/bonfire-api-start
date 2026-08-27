package handler

import (
	"bonfire-api/internal/channel"
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/pkg/ptr"
	"net/http"

	"github.com/google/uuid"
)

type Channel struct {
	service ChannelService
	bind    *httpio.Bind
}

func NewChannel(service ChannelService, bind *httpio.Bind) *Channel {
	return &Channel{
		service: service,
		bind:    bind,
	}
}

type CreateGroupRequest struct {
	PeerIDs []uuid.UUID `json:"peerIds" validate:"required"`
}

func (h *Channel) CreateGroup(w http.ResponseWriter, r *http.Request) error {
	actorID, err := httpio.CtxGetUserID(r.Context())
	if err != nil {
		return err
	}

	var req CreateGroupRequest
	err = h.bind.JSON(w, r, &req)
	if err != nil {
		return err
	}

	err = h.service.CreateGroup(r.Context(), actorID.UUID(), req.PeerIDs)
	if err != nil {
		return err
	}

	httpio.RespondNoContent(w)
	return nil
}

type ChannelGetPath struct {
	ChannelID uuid.UUID `path:"channelId" validate:"required,uuid"`
}

type ChannelGetQuery struct {
	MessageID *uuid.UUID `form:"messageID,omitempty" validate:"omitempty,uuid"`
}

type GetChannelResponse struct {
	Channel  *channel.Channel      `json:"channel"`
	Members  []channel.MemberView  `json:"members"`
	Messages []channel.MessageView `json:"messages"`
}

func (h *Channel) Get(w http.ResponseWriter, r *http.Request) error {
	actorID, err := httpio.CtxGetUserID(r.Context())
	if err != nil {
		return err
	}

	var path ChannelGetPath
	if err := h.bind.Path(r, &path); err != nil {
		return err
	}

	var query ChannelGetQuery
	if err := h.bind.Query(r, &query); err != nil {
		return err
	}

	ch, members, messages, err := h.service.Get(
		r.Context(),
		actorID.UUID(),
		path.ChannelID,
		ptr.From(query.MessageID),
	)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, GetChannelResponse{
		Channel:  ch,
		Members:  members,
		Messages: messages,
	})
	return nil
}

type UpdateGroupPath struct {
	ChannelID uuid.UUID `path:"channelId" validate:"required,uuid"`
}

type UpdateGroupRequest struct {
	Name    *string `json:"name,omitempty" mod:"text" validate:"omitempty,min=1,max=100"`
	IconURL *string `json:"iconUrl,omitempty" mod:"text" validate:"omitempty,url"`
}

func (h *Channel) UpdateGroup(w http.ResponseWriter, r *http.Request) error {
	actorID, err := httpio.CtxGetUserID(r.Context())
	if err != nil {
		return err
	}

	var path UpdateGroupPath
	if err := h.bind.Path(r, &path); err != nil {
		return err
	}

	var req UpdateGroupRequest
	if err := h.bind.JSON(w, r, &req); err != nil {
		return err
	}

	ch, err := h.service.UpdateGroup(
		r.Context(),
		actorID.UUID(),
		path.ChannelID,
		req.Name,
		req.IconURL,
	)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, ch)
	return nil
}
