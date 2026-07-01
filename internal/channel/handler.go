package channel

import (
	"net/http"

	"bonfire-api/internal/apperr"
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/validator"

	"github.com/google/uuid"
)

type Handler struct {
	service   *Service
	validator *validator.Validator
}

func NewHandler(service *Service, validator *validator.Validator) *Handler {
	return &Handler{service: service, validator: validator}
}

// --- CreateDM JSON Request Schema ---
type CreateDMReq struct {
	PeerID uuid.UUID `json:"peer_id" validate:"required"`
}

func (h *Handler) CreateDM(w http.ResponseWriter, r *http.Request) error {
	actorID, err := httpio.GetCtxUserID(r.Context())
	if err != nil {
		return apperr.NewUnauthorized(err)
	}

	body, err := httpio.BindJSON[CreateDMReq](w, r, h.validator)
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
