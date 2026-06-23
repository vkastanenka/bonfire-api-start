package auth

import (
	"bonfire-api/internal/httpio"
	"net/http"
)

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) error {
	// Bind JSON
	reqData, err := httpio.BindJSON[RegisterReq](w, r, h.val)
	if err != nil {
		return err
	}

	// Register user
	data, err := h.service.Register(r.Context(), RegisterParams{
		Email:       reqData.Email,
		Username:    reqData.Username,
		DisplayName: reqData.DisplayName,
		Password:    reqData.Password,
	})
	if err != nil {
		return err
	}

	// Respond
	httpio.RespondCreated(w, r, data, RegisterOk)

	return nil
}
