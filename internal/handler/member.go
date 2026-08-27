package handler

import (
	"net/http"

	"bonfire-api/internal/channel"
	"bonfire-api/internal/httpio"

	"github.com/google/uuid"
)

type MemberHandler struct {
	service channel.MemberService
	bind    *httpio.Bind
}

func NewMemberHandler(service channel.MemberService, bind *httpio.Bind) *MemberHandler {
	return &MemberHandler{
		service: service,
		bind:    bind,
	}
}

type ChannelPath struct {
	ChannelID uuid.UUID `path:"channelId" validate:"required,uuid"`
}

type AddMembersRequest struct {
	MemberIDs []uuid.UUID `json:"memberIds" validate:"required,min=1,max=100,dive,uuid"`
}

func (h *MemberHandler) AddMembers(w http.ResponseWriter, r *http.Request) error {
	actorID, err := httpio.CtxGetUserID(r.Context())
	if err != nil {
		return err
	}

	var path ChannelPath
	if err := h.bind.Path(r, &path); err != nil {
		return err
	}

	var req AddMembersRequest
	if err := h.bind.JSON(w, r, &req); err != nil {
		return err
	}

	if err := h.service.AddMembers(r.Context(), actorID.UUID(), path.ChannelID, req.MemberIDs); err != nil {
		return err
	}

	httpio.RespondNoContent(w)
	return nil
}

func (h *MemberHandler) CloseDirect(w http.ResponseWriter, r *http.Request) error {
	actorID, err := httpio.CtxGetUserID(r.Context())
	if err != nil {
		return err
	}

	var path ChannelPath
	if err := h.bind.Path(r, &path); err != nil {
		return err
	}

	if err := h.service.CloseDirect(r.Context(), actorID.UUID(), path.ChannelID); err != nil {
		return err
	}

	httpio.RespondNoContent(w)
	return nil
}

type UpdateLastReadMessageRequest struct {
	LastReadMessageID uuid.UUID `json:"lastReadMessageId" validate:"required,uuid"`
}

func (h *MemberHandler) UpdateLastReadMessage(w http.ResponseWriter, r *http.Request) error {
	actorID, err := httpio.CtxGetUserID(r.Context())
	if err != nil {
		return err
	}

	var path ChannelPath
	if err := h.bind.Path(r, &path); err != nil {
		return err
	}

	var req UpdateLastReadMessageRequest
	if err := h.bind.JSON(w, r, &req); err != nil {
		return err
	}

	member, err := h.service.UpdateLastReadMessage(r.Context(), actorID.UUID(), path.ChannelID, req.LastReadMessageID)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, member)
	return nil
}

type UpdatePinnedAtRequest struct {
	IsPinned bool `json:"isPinned"`
}

func (h *MemberHandler) UpdatePinnedAt(w http.ResponseWriter, r *http.Request) error {
	actorID, err := httpio.CtxGetUserID(r.Context())
	if err != nil {
		return err
	}

	var path ChannelPath
	if err := h.bind.Path(r, &path); err != nil {
		return err
	}

	var req UpdatePinnedAtRequest
	if err := h.bind.JSON(w, r, &req); err != nil {
		return err
	}

	member, err := h.service.UpdatePinnedAt(r.Context(), actorID.UUID(), path.ChannelID, req.IsPinned)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, member)
	return nil
}

type UpdateMutedUntilRequest struct {
	Duration *int `json:"duration,omitempty"`
}

func (h *MemberHandler) UpdateMutedUntil(w http.ResponseWriter, r *http.Request) error {
	actorID, err := httpio.CtxGetUserID(r.Context())
	if err != nil {
		return err
	}

	var path ChannelPath
	if err := h.bind.Path(r, &path); err != nil {
		return err
	}

	var req UpdateMutedUntilRequest
	if err := h.bind.JSON(w, r, &req); err != nil {
		return err
	}

	member, err := h.service.UpdateMutedUntil(r.Context(), actorID.UUID(), path.ChannelID, req.Duration)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, member)
	return nil
}

func (h *MemberHandler) LeaveGroup(w http.ResponseWriter, r *http.Request) error {
	actorID, err := httpio.CtxGetUserID(r.Context())
	if err != nil {
		return err
	}

	var path ChannelPath
	if err := h.bind.Path(r, &path); err != nil {
		return err
	}

	if err := h.service.LeaveGroup(r.Context(), actorID.UUID(), path.ChannelID); err != nil {
		return err
	}

	httpio.RespondNoContent(w)
	return nil
}
