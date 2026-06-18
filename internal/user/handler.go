package user

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/validator"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	service *UserService
	val     *validator.Validator
}

func NewHandler(service *UserService, val *validator.Validator) *Handler {
	return &Handler{service: service, val: val}
}

// Ping confirms the auth routes are available
func (h *Handler) Ping(w http.ResponseWriter, r *http.Request) error {
	httpio.RespondJSON(w, http.StatusOK, map[string]string{
		"status": "healthy",
	})

	return nil
}

// Request structs for validation
type EmailRequest struct {
	Email string `validate:"required,email,max=255"`
}

// GetByID handles GET /users/{userID}
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) error {
	idStr := chi.URLParam(r, "userID")
	userID, err := uuid.Parse(idStr)
	if err != nil {
		return apperr.New(apperr.CodeInvalidInput, "Invalid UUID format")
	}

	user, err := h.service.GetUserByID(r.Context(), userID)
	if err != nil {
		return err
	}

	httpio.RespondJSON(w, http.StatusOK, user)

	return nil
}

// GetByEmail handles GET /users?email=...
func (h *Handler) GetByEmail(w http.ResponseWriter, r *http.Request) error {
	email := r.URL.Query().Get("email")
	req := EmailRequest{Email: email}

	if err := h.val.ValidateStruct(&req); err != nil {
		return err
	}

	user, err := h.service.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		return err
	}

	httpio.RespondJSON(w, http.StatusOK, user)

	return nil
}

// DeleteByEmail handles DELETE /users?email=...
func (h *Handler) DeleteByEmail(w http.ResponseWriter, r *http.Request) error {
	email := r.URL.Query().Get("email")
	req := EmailRequest{Email: email}

	if err := h.val.ValidateStruct(&req); err != nil {
		return err
	}

	user, err := h.service.DeleteUserByEmail(r.Context(), req.Email)
	if err != nil {
		return err
	}

	// Returning the deleted user object is often useful for confirmation
	httpio.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "User deleted successfully",
		"user":    user,
	})

	return nil
}
