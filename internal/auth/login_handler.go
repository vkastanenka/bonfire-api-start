package auth

import (
	"bonfire-api/internal/httpio"
	"net/http"
)

// Login handles user login
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) error {
	// Get JSON
	req, err := httpio.BindJSON[LoginReq](w, r, h.validator)
	if err != nil {
		return err
	}

	// Get client meta
	clientMeta := httpio.GetClientMeta(r)

	// Login user, get tokens
	tokens, err := h.service.Login(r.Context(), LoginParams{
		Email:    req.Email,
		Password: req.Password,
		Meta:     clientMeta,
	})
	if err != nil {
		return err
	}

	// Repond with tokens
	httpio.SetRefreshTokenCookie(w, tokens.RefreshToken)
	httpio.RespondOK(w, r, LoginRes{AccessToken: tokens.AccessToken}, LoginOK)

	return nil
}
