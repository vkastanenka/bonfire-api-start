package message

import (
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/validator"
	"net/http"

	"github.com/google/uuid"
)

type Handler struct {
	service   *Service
	validator *validator.Validator
}

func NewHandler(service *Service, validator *validator.Validator) *Handler {
	return &Handler{service: service, validator: validator}
}

// --- Send Message ---

type SendMessageReq struct {
	Content string `json:"content" validate:"required,max=2000"`
}

type SendMessagePath struct {
	ConversationID uuid.UUID `path:"conversation_id" validate:"required"`
}

func (h *Handler) Send(w http.ResponseWriter, r *http.Request) error {
	userID, err := httpio.GetCtxUserID(r.Context())
	if err != nil {
		return err
	}

	path, err := httpio.BindPath[SendMessagePath](r, h.validator)
	if err != nil {
		return err
	}

	req, err := httpio.BindJSON[SendMessageReq](w, r, h.validator)
	if err != nil {
		return err
	}

	view, err := h.service.Send(r.Context(), SendParams{
		UserID:         userID,
		ConversationID: path.ConversationID,
		Content:        req.Content,
	})
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, view, "message sent")
	return nil
}

// --- List Messages ---

type ListMessagesPath struct {
	ConversationID uuid.UUID `path:"conversation_id" validate:"required"`
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) error {
	path, err := httpio.BindPath[ListMessagesPath](r, h.validator)
	if err != nil {
		return err
	}

	views, err := h.service.List(r.Context(), path.ConversationID)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, views, "")
	return nil
}
