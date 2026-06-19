package auth

import (
	"bonfire-api/internal/httpio"
	"net/http"
)

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) error {
	req, err := httpio.BindJSON[RegisterRequest](w, r, h.val)
	if err != nil {
		return err
	}

	// Register user
	user, profile, err := h.service.Register(r.Context(), RegisterInput{
		Email:       req.Email,
		Username:    req.Username,
		DisplayName: req.DisplayName,
		Password:    req.Password,
	})
	if err != nil {
		return err
	}

	// Format response data
	data := RegisterResponse{
		User:        user,
		UserProfile: profile,
	}

	// Respond
	httpio.RespondCreated(w, data, "User successfully registered.")

	return nil
}
