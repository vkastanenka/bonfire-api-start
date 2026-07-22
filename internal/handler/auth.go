package handler

import (
	"bonfire-api/internal/auth"
	"bonfire-api/internal/httpio"
	"net/http"
)

type AuthHandler struct {
	service AuthService
}

func NewAuthHandler() *AuthHandler {
	return &AuthHandler{}
}

type ForgotPasswordRequest struct {
	Email string `json:"email" mod:"email" validate:"identity_email"`
}

func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) error {
	req, err := httpio.BindJSON[ForgotPasswordRequest](nil, w, r)
	if err != nil {
		return err
	}

	if err := h.service.ForgotPassword(r.Context(), req.Email); err != nil {
		return err
	}

	httpio.RespondNoContent(w)
	return nil
}

type LoginRequest struct {
	Email    string `json:"email" mod:"email" validate:"identity_email"`
	Password string `json:"password" validate:"identity_password"`
}

type LoginResponse struct {
	AccessToken string `json:"access_token"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) error {
	req, err := httpio.BindJSON[LoginRequest](nil, w, r)
	if err != nil {
		return err
	}

	clientMeta, err := httpio.GetClientMeta(r.Context())
	if err != nil {
		return err
	}

	data, err := h.service.Login(r.Context(), auth.LoginParams{
		Email:      req.Email,
		Password:   req.Password,
		ClientMeta: clientMeta,
	})
	if err != nil {
		return err
	}

	httpio.SetCookieRefreshToken(w, httpio.SetCookieRefreshTokenParams{
		Token:   data.RefreshToken,
		Expires: data.RefreshTokenExpiresAt,
	})
	httpio.RespondOK(w, r, LoginResponse{AccessToken: data.AccessToken})

	return nil
}

type RefreshResponse struct {
	AccessToken string `json:"access_token"`
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) error {
	refreshToken, err := httpio.GetCookieRefreshToken(r)
	if err != nil {
		// return apperr.NewUnauthorized(err, "Missing refresh token, please log in.")
	}

	data, err := h.service.Refresh(r.Context(), auth.RefreshParams{
		RefreshToken: refreshToken,
	})
	if err != nil {
		return err
	}

	httpio.SetCookieRefreshToken(w, httpio.SetCookieRefreshTokenParams{
		Token:   data.RefreshToken,
		Expires: data.RefreshTokenExpiresAt,
	})
	httpio.RespondOK(w, r, RefreshResponse{AccessToken: data.AccessToken})

	return nil
}

type RegisterRequest struct {
	Email       string  `json:"email" mod:"email" validate:"identity_email"`
	Username    string  `json:"username" mod:"text" validate:"identity_username"`
	DisplayName *string `json:"display_name" mod:"text" validate:"profile_display_name"`
	Password    string  `json:"password" validate:"identity_password"`
}

type RegisterResponse struct {
	AccessToken string `json:"access_token"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) error {
	req, err := httpio.BindJSON[RegisterRequest](nil, w, r)
	if err != nil {
		return err
	}

	clientMeta, err := httpio.GetClientMeta(r.Context())
	if err != nil {
		return err
	}

	data, err := h.service.Register(r.Context(), auth.RegisterParams{
		Email:       req.Email,
		Username:    req.Username,
		DisplayName: req.DisplayName,
		Password:    req.Password,
		ClientMeta:  clientMeta,
	})
	if err != nil {
		return err
	}

	httpio.SetCookieRefreshToken(w, httpio.SetCookieRefreshTokenParams{
		Token:   data.RefreshToken,
		Expires: data.RefreshTokenExpiresAt,
	})
	httpio.RespondCreated(w, r, RegisterResponse{AccessToken: data.AccessToken})
	return nil
}

func (h *AuthHandler) ResendVerify(w http.ResponseWriter, r *http.Request) error {
	userID, err := httpio.GetCtxUserID(r.Context())
	if err != nil {
		return err
	}

	if err := h.service.ResendVerify(r.Context(), userID); err != nil {
		return err
	}

	httpio.RespondNoContent(w)
	return nil
}

type ResetPasswordRequest struct {
	Token    string `json:"token" validate:"required"`
	Password string `json:"password" validate:"identity_password"`
}

type ResetPasswordResponse struct {
	AccessToken string `json:"access_token"`
}

func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) error {
	req, err := httpio.BindJSON[ResetPasswordRequest](nil, w, r)
	if err != nil {
		return err
	}

	clientMeta, err := httpio.GetClientMeta(r.Context())
	if err != nil {
		return err
	}

	data, err := h.service.ResetPassword(r.Context(), auth.ResetPasswordParams{
		Token:      req.Token,
		Password:   req.Password,
		ClientMeta: clientMeta,
	})
	if err != nil {
		return err
	}

	httpio.SetCookieRefreshToken(w, httpio.SetCookieRefreshTokenParams{
		Token:   data.RefreshToken,
		Expires: data.RefreshTokenExpiresAt,
	})
	httpio.RespondOK(w, r, RegisterResponse{AccessToken: data.AccessToken})
	return nil
}

type VerifyEmailRequest struct {
	Token string `json:"token" validate:"required"`
}

func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) error {
	req, err := httpio.BindJSON[VerifyEmailRequest](nil, w, r)
	if err != nil {
		return err
	}

	if err := h.service.VerifyEmail(r.Context(), req.Token); err != nil {
		return err
	}

	httpio.RespondNoContent(w)
	return nil
}

type WSTicketResponse struct {
	Ticket string `json:"ticket"`
}

func (h *AuthHandler) WSTicket(w http.ResponseWriter, r *http.Request) error {
	claims, err := httpio.GetCtxClaims(r.Context())
	if err != nil {
		return err
	}

	ticket, err := h.service.WSTicket(r.Context(), auth.WSTicketData{
		UserID: claims.UserID,
	})
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, WSTicketResponse{Ticket: ticket.String()})
	return nil
}
