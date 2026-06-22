package user_profile

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/httpio"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Ping confirms the auth routes are available
func (h *Handler) Ping(w http.ResponseWriter, r *http.Request) error {
	httpio.RespondOK(w, r, map[string]string{
		"status": "healthy",
	}, "Ping OK.")

	return nil
}

// GetByUserID handles GET /user_profiles/{userID}
func (h *Handler) GetByUserID(w http.ResponseWriter, r *http.Request) error {
	idStr := chi.URLParam(r, "id")
	userID, err := uuid.Parse(idStr)
	if err != nil {
		return apperr.New(apperr.CodeInvalidInput, "Invalid UUID format")
	}

	user_profile, err := h.service.GetByUserID(r.Context(), userID)
	if err != nil {
		return err
	}

	httpio.RespondJSON(w, r, http.StatusOK, user_profile)

	return nil
}
