package handler

import (
	"context"
	"net/http"

	"bonfire-api/internal/channel"
	"bonfire-api/internal/httpio"

	"github.com/google/uuid"
)

type ChannelService interface {
	CreateChannel(ctx context.Context, chType channel.Type, creatorID uuid.UUID, recipientIDs []uuid.UUID, name *string, iconURL *string) (*channel.Channel, error)
	PostMessage(ctx context.Context, channelID, authorID uuid.UUID, rawContent string, replyToID *uuid.UUID) (*channel.Message, error)
	EditMessage(ctx context.Context, messageID, authorID uuid.UUID, newRawContent string) (*channel.Message, error)
	DeleteMessage(ctx context.Context, messageID, actorID uuid.UUID) error
	ListMessages(ctx context.Context, channelID, userID uuid.UUID, before *uuid.UUID, limit int) ([]channel.Message, error)
}

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

// --- Requests ---

type CreateChannelRequest struct {
	Type         channel.Type `json:"type" validate:"required"`
	RecipientIDs []uuid.UUID  `json:"recipientIds" validate:"required,min=1"`
	Name         *string      `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
	IconURL      *string      `json:"iconUrl,omitempty" validate:"omitempty,url"`
}

type ChannelPath struct {
	ChannelID uuid.UUID `path:"channelId" validate:"required,uuid"`
}

type MessagePath struct {
	ChannelID uuid.UUID `path:"channelId" validate:"required,uuid"`
	MessageID uuid.UUID `path:"messageId" validate:"required,uuid"`
}

type PostMessageRequest struct {
	Content   string     `json:"content" validate:"required,min=1,max=2000"`
	ReplyToID *uuid.UUID `json:"replyToId,omitempty" validate:"omitempty,uuid"`
}

type EditMessageRequest struct {
	Content string `json:"content" validate:"required,min=1,max=2000"`
}

type ListMessagesQuery struct {
	Before *uuid.UUID `query:"before" validate:"omitempty,uuid"`
	Limit  int        `query:"limit" validate:"omitempty,min=1,max=100"`
}

// --- Handler Methods ---

func (h *Channel) Create(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	creatorID, err := httpio.CtxGetUserID(ctx)
	if err != nil {
		return err
	}

	var req CreateChannelRequest
	if err := h.bind.JSON(w, r, &req); err != nil {
		return err
	}

	ch, err := h.service.CreateChannel(ctx, req.Type, creatorID, req.RecipientIDs, req.Name, req.IconURL)
	if err != nil {
		return err
	}

	httpio.RespondCreated(w, r, ToChannelResponse(*ch))
	return nil
}

func (h *Channel) PostMessage(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	userID, err := httpio.CtxGetUserID(ctx)
	if err != nil {
		return err
	}

	var path ChannelPath
	if err := h.bind.Path(r, &path); err != nil {
		return err
	}

	var req PostMessageRequest
	if err := h.bind.JSON(w, r, &req); err != nil {
		return err
	}

	msg, err := h.service.PostMessage(ctx, path.ChannelID, userID, req.Content, req.ReplyToID)
	if err != nil {
		return err
	}

	httpio.RespondCreated(w, r, ToMessageResponse(*msg))
	return nil
}

func (h *Channel) EditMessage(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	userID, err := httpio.CtxGetUserID(ctx)
	if err != nil {
		return err
	}

	var path MessagePath
	if err := h.bind.Path(r, &path); err != nil {
		return err
	}

	var req EditMessageRequest
	if err := h.bind.JSON(w, r, &req); err != nil {
		return err
	}

	msg, err := h.service.EditMessage(ctx, path.MessageID, userID, req.Content)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, ToMessageResponse(*msg))
	return nil
}

func (h *Channel) DeleteMessage(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	userID, err := httpio.CtxGetUserID(ctx)
	if err != nil {
		return err
	}

	var path MessagePath
	if err := h.bind.Path(r, &path); err != nil {
		return err
	}

	if err := h.service.DeleteMessage(ctx, path.MessageID, userID); err != nil {
		return err
	}

	httpio.RespondNoContent(w)
	return nil
}

func (h *Channel) ListMessages(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	userID, err := httpio.CtxGetUserID(ctx)
	if err != nil {
		return err
	}

	var path ChannelPath
	if err := h.bind.Path(r, &path); err != nil {
		return err
	}

	var query ListMessagesQuery
	if err := h.bind.Query(r, &query); err != nil {
		return err
	}

	messages, err := h.service.ListMessages(ctx, path.ChannelID, userID, query.Before, query.Limit)
	if err != nil {
		return err
	}

	responses := make([]MessageResponse, len(messages))
	for i, msg := range messages {
		responses[i] = ToMessageResponse(msg)
	}

	httpio.RespondOK(w, r, responses)
	return nil
}
