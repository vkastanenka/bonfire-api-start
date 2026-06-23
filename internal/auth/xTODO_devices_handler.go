package auth

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/httpio"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

/*
TODO
*/

// GetDevices returns all active sessions for the logged-in user
func (h *Handler) GetDevices(w http.ResponseWriter, r *http.Request) error {
	userID, err := GetUserIDFromContext(r.Context())
	if err != nil {
		return apperr.New(apperr.CodeUnauthenticated, "Missing user identity in context.")
	}

	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		return apperr.New(apperr.CodeUnauthenticated, "Missing refresh token.")
	}

	devices, err := h.service.GetDevices(r.Context(), userID, cookie.Value)
	if err != nil {
		return err
	}

	httpio.RespondJSON(w, r, http.StatusOK, map[string]interface{}{
		"devices": devices,
	})
	return nil
}

// RevokeDevice logs the user out of a specific session
func (h *Handler) RevokeDevice(w http.ResponseWriter, r *http.Request) error {
	userID, err := GetUserIDFromContext(r.Context())
	if err != nil {
		return apperr.New(apperr.CodeUnauthenticated, "Missing user identity in context.")
	}

	// Extract session ID from URL (e.g., /auth/devices/{id})
	sessionIDStr := chi.URLParam(r, "id")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		return apperr.New(apperr.CodeInvalidInput, "Invalid device ID format.")
	}

	if err := h.service.RevokeDevice(r.Context(), userID, sessionID); err != nil {
		return err
	}

	httpio.RespondJSON(w, r, http.StatusOK, map[string]string{
		"message": "Successfully logged out of device.",
	})
	return nil
}

// RevokeAllOtherDevices logs the user out of all devices EXCEPT the current one
func (h *Handler) RevokeAllOtherDevices(w http.ResponseWriter, r *http.Request) error {
	userID, err := GetUserIDFromContext(r.Context())
	if err != nil {
		return apperr.New(apperr.CodeUnauthenticated, "Missing user identity in context.")
	}

	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		return apperr.New(apperr.CodeUnauthenticated, "Missing refresh token.")
	}

	if err := h.service.RevokeAllOtherDevices(r.Context(), userID, cookie.Value); err != nil {
		return err
	}

	httpio.RespondJSON(w, r, http.StatusOK, map[string]string{
		"message": "Successfully logged out of all other known devices.",
	})
	return nil
}
