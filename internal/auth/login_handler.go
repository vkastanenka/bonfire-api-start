package auth

import (
	"bonfire-api/internal/httpio"
	"net/http"
)

// Login handles user login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) error {
	// Get JSON
	req, err := httpio.BindJSON[LoginReq](w, r, h.val)
	if err != nil {
		return err
	}

	// Get client meta
	clientMeta := httpio.GetClientMeta(r)

	// Login user, get tokens
	tokens, err := h.service.Login(r.Context(), LoginParams{
		Email:     req.Email,
		Password:  req.Password,
		UserAgent: clientMeta.UserAgent,
		ClientIP:  clientMeta.IP,
	})
	if err != nil {
		return err
	}

	// Use tokens
	httpio.SetRefreshTokenCookie(w, tokens.RefreshToken)
	httpio.RespondOK(w, LoginRes{AccessToken: tokens.AccessToken}, LoginOk)

	return nil
}
