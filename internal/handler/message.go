package handler

import (
	"net/http"
	"time"

	"bonfire-api/internal/channel"
	"bonfire-api/internal/httpio"

	"github.com/google/uuid"
)

type MessageHandler struct {
	service *channel.MessageService
	bind    *httpio.Bind
}

func NewMessageHandler(service *channel.MessageService, bind *httpio.Bind) *MessageHandler {
	return &MessageHandler{
		service: service,
		bind:    bind,
	}
}

type MessagePath struct {
	ChannelID uuid.UUID `path:"channelId" validate:"required,uuid"`
	MessageID uuid.UUID `path:"messageId" validate:"required,uuid"`
}

type ChannelMessagePath struct {
	ChannelID uuid.UUID `path:"channelId" validate:"required,uuid"`
}

type CreateMessageRequest struct {
	Content          *string    `json:"content"`
	ReplyToMessageId *uuid.UUID `json:"replyToMessageId" validate:"omitempty,uuid"`
	ForwardMessageId *uuid.UUID `json:"forwardMessageId" validate:"omitempty,uuid"`
	ForwardChannelId *uuid.UUID `json:"forwardChannelId" validate:"omitempty,uuid"`
}

func (h *MessageHandler) Create(w http.ResponseWriter, r *http.Request) error {
	actorID, err := httpio.CtxGetUserID(r.Context())
	if err != nil {
		return err
	}

	var path ChannelMessagePath
	if err := h.bind.Path(r, &path); err != nil {
		return err
	}

	var req CreateMessageRequest
	if err := h.bind.JSON(w, r, &req); err != nil {
		return err
	}

	msgView, err := h.service.Create(
		r.Context(),
		actorID.UUID(),
		path.ChannelID,
		req.Content,
		req.ReplyToMessageId,
		req.ForwardMessageId,
		req.ForwardChannelId,
	)
	if err != nil {
		return err
	}

	httpio.RespondCreated(w, r, msgView)
	return nil
}

type ListMessagesQuery struct {
	CursorID  uuid.UUID `query:"cursorId" validate:"required,uuid"`
	Direction string    `query:"direction" validate:"required,oneof=around before after"`
}

func (h *MessageHandler) List(w http.ResponseWriter, r *http.Request) error {
	actorID, err := httpio.CtxGetUserID(r.Context())
	if err != nil {
		return err
	}

	var path ChannelMessagePath
	if err := h.bind.Path(r, &path); err != nil {
		return err
	}

	var q ListMessagesQuery
	if err := h.bind.Query(r, &q); err != nil {
		return err
	}

	var (
		messages []channel.MessageView
		listErr  error
	)

	switch q.Direction {
	case "around":
		messages, listErr = h.service.ListAround(r.Context(), actorID.UUID(), path.ChannelID, q.CursorID)
	case "before":
		messages, listErr = h.service.ListBefore(r.Context(), actorID.UUID(), path.ChannelID, q.CursorID)
	case "after":
		messages, listErr = h.service.ListAfter(r.Context(), actorID.UUID(), path.ChannelID, q.CursorID)
	}

	if listErr != nil {
		return listErr
	}

	httpio.RespondOK(w, r, messages)
	return nil
}

type ListPinnedQuery struct {
	CursorID       *uuid.UUID `query:"cursorId" validate:"omitempty,uuid"`
	CursorPinnedAt *time.Time `query:"cursorPinnedAt" validate:"omitempty"`
}

func (h *MessageHandler) ListPinned(w http.ResponseWriter, r *http.Request) error {
	actorID, err := httpio.CtxGetUserID(r.Context())
	if err != nil {
		return err
	}

	var path ChannelMessagePath
	if err := h.bind.Path(r, &path); err != nil {
		return err
	}

	var q ListPinnedQuery
	if err := h.bind.Query(r, &q); err != nil {
		return err
	}

	pins, err := h.service.ListPinned(r.Context(), actorID.UUID(), path.ChannelID, q.CursorID, q.CursorPinnedAt)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, pins)
	return nil
}

type UpdateMessageContentRequest struct {
	Content string `json:"content" validate:"required"`
}

func (h *MessageHandler) UpdateContent(w http.ResponseWriter, r *http.Request) error {
	actorID, err := httpio.CtxGetUserID(r.Context())
	if err != nil {
		return err
	}

	var path MessagePath
	if err := h.bind.Path(r, &path); err != nil {
		return err
	}

	var req UpdateMessageContentRequest
	if err := h.bind.JSON(w, r, &req); err != nil {
		return err
	}

	msg, err := h.service.UpdateContent(r.Context(), actorID.UUID(), path.ChannelID, path.MessageID, req.Content)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, msg)
	return nil
}

type UpdateMessagePinnedAtRequest struct {
	IsPinned bool `json:"isPinned"`
}

func (h *MessageHandler) UpdatePinnedAt(w http.ResponseWriter, r *http.Request) error {
	actorID, err := httpio.CtxGetUserID(r.Context())
	if err != nil {
		return err
	}

	var path MessagePath
	if err := h.bind.Path(r, &path); err != nil {
		return err
	}

	var req UpdateMessagePinnedAtRequest
	if err := h.bind.JSON(w, r, &req); err != nil {
		return err
	}

	msg, err := h.service.UpdatePinnedAt(r.Context(), actorID.UUID(), path.ChannelID, path.MessageID, req.IsPinned)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, msg)
	return nil
}

func (h *MessageHandler) Delete(w http.ResponseWriter, r *http.Request) error {
	actorID, err := httpio.CtxGetUserID(r.Context())
	if err != nil {
		return err
	}

	var path MessagePath
	if err := h.bind.Path(r, &path); err != nil {
		return err
	}

	if err := h.service.Delete(r.Context(), actorID.UUID(), path.ChannelID, path.MessageID); err != nil {
		return err
	}

	httpio.RespondNoContent(w)
	return nil
}

type ToggleReactionRequest struct {
	Emoji string `json:"emoji" validate:"required"`
}

func (h *MessageHandler) ToggleReaction(w http.ResponseWriter, r *http.Request) error {
	actorID, err := httpio.CtxGetUserID(r.Context())
	if err != nil {
		return err
	}

	var path MessagePath
	if err := h.bind.Path(r, &path); err != nil {
		return err
	}

	var req ToggleReactionRequest
	if err := h.bind.JSON(w, r, &req); err != nil {
		return err
	}

	count, err := h.service.ToggleReaction(r.Context(), actorID.UUID(), path.ChannelID, path.MessageID, req.Emoji)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, count)
	return nil
}
