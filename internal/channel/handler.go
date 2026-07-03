package channel

import (
	"net/http"

	"bonfire-api/internal/apperr"
	"bonfire-api/internal/httpio"

	"github.com/google/uuid"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

type CreateDMReq struct {
	PeerID uuid.UUID `json:"peer_id" validate:"required"`
}

func (h *Handler) CreateDM(w http.ResponseWriter, r *http.Request) error {
	actorID, err := httpio.GetCtxUserID(r.Context())
	if err != nil {
		return apperr.NewUnauthorized(err, "")
	}

	body, err := httpio.BindJSON[CreateDMReq](w, r)
	if err != nil {
		return err
	}

	view, err := h.service.FindOrCreateDM(r.Context(), CreateDMParams{
		ActorID: actorID,
		PeerID:  body.PeerID,
	})
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, view, "channel initialized successfully")
	return nil
}
