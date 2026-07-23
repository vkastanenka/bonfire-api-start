package handler

import (
	"bonfire-api/internal/auth"
	"bonfire-api/internal/errs"
	"bonfire-api/internal/httpio"
	"net/http"
)

type AuthHandler struct {
	service AuthService
	bind    *httpio.Bind
}

func NewAuthHandler(service AuthService, bind *httpio.Bind) *AuthHandler {
	return &AuthHandler{
		service: service,
		bind:    bind,
	}
}

type ForgotPasswordRequest struct {
	Email string `json:"email" mod:"email" validate:"required,email,max=255"`
}

func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) error {
	var req ForgotPasswordRequest
	err := h.bind.JSON(w, r, &req)
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
	Email    string `json:"email" mod:"email" validate:"required,email,max=255"`
	Password string `json:"password" validate:"required,min=12,max=255"`
}

type LoginResponse struct {
	AccessToken string `json:"access_token"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) error {
	var req LoginRequest
	err := h.bind.JSON(w, r, &req)
	if err != nil {
		return err
	}

	clientMeta, err := httpio.CtxGetMeta(r.Context())
	if err != nil {
		return err
	}

	data, err := h.service.Login(r.Context(), auth.LoginParams{
		Email:     req.Email,
		Password:  req.Password,
		IP:        clientMeta.IP,
		UserAgent: clientMeta.UserAgent,
		OS:        clientMeta.OS,
		Browser:   clientMeta.Browser,
	})
	if err != nil {
		return err
	}

	httpio.CookieSetRefreshToken(w, data.RefreshToken, data.RefreshTokenExpiresAt)
	httpio.RespondOK(w, r, LoginResponse{AccessToken: data.AccessToken})
	return nil
}

type RefreshResponse struct {
	AccessToken string `json:"access_token"`
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) error {
	refreshToken, err := httpio.CookieGetRefreshToken(r)
	if err != nil {
		return errs.Unauthenticated("Missing refresh token, please log in.").Wrap(err)
	}

	data, err := h.service.Refresh(r.Context(), auth.RefreshParams{
		RefreshToken: refreshToken,
	})
	if err != nil {
		return err
	}

	httpio.CookieSetRefreshToken(w, data.RefreshToken, data.RefreshTokenExpiresAt)
	httpio.RespondOK(w, r, RefreshResponse{AccessToken: data.AccessToken})

	return nil
}

type RegisterRequest struct {
	Email       string  `json:"email" mod:"email" validate:"required,email,max=255"`
	Username    string  `json:"username" mod:"text" validate:"required,alphanum,min=3,max=32"`
	DisplayName *string `json:"displayName,omitempty" mod:"text" validate:"min=3,max=32"`
	Password    string  `json:"password" validate:"required,min=12,max=255"`
}

type RegisterResponse struct {
	AccessToken string `json:"access_token"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) error {
	var req RegisterRequest
	err := h.bind.JSON(w, r, &req)
	if err != nil {
		return err
	}

	clientMeta, err := httpio.CtxGetMeta(r.Context())
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

	httpio.CookieSetRefreshToken(w, data.RefreshToken, data.RefreshTokenExpiresAt)
	httpio.RespondCreated(w, r, RegisterResponse{AccessToken: data.AccessToken})
	return nil
}

func (h *AuthHandler) ResendVerify(w http.ResponseWriter, r *http.Request) error {
	userID, err := httpio.CtxGetUserID(r.Context())
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
	Token    string `json:"token" validate:"required,token"`
	Password string `json:"password" validate:"required,min=12,max=255"`
}

type ResetPasswordResponse struct {
	AccessToken string `json:"access_token"`
}

func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) error {
	var req ResetPasswordRequest
	err := h.bind.JSON(w, r, &req)
	if err != nil {
		return err
	}

	clientMeta, err := httpio.CtxGetMeta(r.Context())
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

	httpio.CookieSetRefreshToken(w, data.RefreshToken, data.RefreshTokenExpiresAt)
	httpio.RespondOK(w, r, RegisterResponse{AccessToken: data.AccessToken})
	return nil
}

type VerifyEmailRequest struct {
	Token string `json:"token" validate:"required,token"`
}

func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) error {
	var req VerifyEmailRequest
	err := h.bind.JSON(w, r, &req)
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
	claims, err := httpio.CtxGetClaims(r.Context())
	if err != nil {
		return err
	}

	ticket, err := h.service.WSTicket(r.Context(), claims.UserID)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, WSTicketResponse{Ticket: ticket.String()})
	return nil
}
