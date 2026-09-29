package handler

import (
	"bonfire-api/internal/auth"
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/pkg/errs"
	"bonfire-api/internal/user"
	"net/http"

	"github.com/google/uuid"
)

type AccessTokenResponse struct {
	AccessToken string `json:"accessToken"`
}

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
	Email string `json:"email" mod:"email" validate:"required,email_spec"`
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

type ResetPasswordRequest struct {
	Token    string `json:"token" validate:"required,token"`
	Password string `json:"password" validate:"required,user_password"`
}

func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) error {
	var req ResetPasswordRequest
	err := h.bind.JSON(w, r, &req)
	if err != nil {
		return err
	}

	data, err := h.service.ResetPassword(r.Context(), auth.ResetPasswordParams{
		Token:    req.Token,
		Password: req.Password,
	})
	if err != nil {
		return err
	}

	httpio.CookieSetRefreshToken(w, data.RefreshToken, data.RefreshTokenExpiresAt)
	httpio.RespondOK(w, r, AccessTokenResponse{AccessToken: data.AccessToken})
	return nil
}

type LoginRequest struct {
	Email    string `json:"email" mod:"email" validate:"required,email_spec"`
	Password string `json:"password" validate:"required,user_password"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) error {
	var req LoginRequest
	err := h.bind.JSON(w, r, &req)
	if err != nil {
		return err
	}

	data, err := h.service.Login(r.Context(), auth.LoginParams{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		return err
	}

	httpio.CookieSetRefreshToken(w, data.RefreshToken, data.RefreshTokenExpiresAt)
	httpio.RespondOK(w, r, AccessTokenResponse{AccessToken: data.AccessToken})
	return nil
}

type RegisterRequest struct {
	Email       string  `json:"email" mod:"email" validate:"required,email_spec"`
	Username    string  `json:"username" mod:"text" validate:"required,user_username"`
	DisplayName *string `json:"displayName,omitempty" mod:"text" validate:"omitempty,user_display_name"`
	Password    string  `json:"password" validate:"required,user_password"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) error {
	var req RegisterRequest
	err := h.bind.JSON(w, r, &req)
	if err != nil {
		return err
	}

	data, err := h.service.Register(r.Context(), auth.RegisterParams{
		Email:       req.Email,
		Username:    req.Username,
		DisplayName: req.DisplayName,
		Password:    req.Password,
	})
	if err != nil {
		return err
	}

	httpio.CookieSetRefreshToken(w, data.RefreshToken, data.RefreshTokenExpiresAt)
	httpio.RespondCreated(w, r, AccessTokenResponse{AccessToken: data.AccessToken})
	return nil
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
	httpio.RespondOK(w, r, AccessTokenResponse{AccessToken: data.AccessToken})
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

	u, err := h.service.VerifyEmail(r.Context(), req.Token)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, user.ParseMe(u))
	return nil
}

func (h *AuthHandler) ResendVerify(w http.ResponseWriter, r *http.Request) error {
	if err := h.service.ResendVerify(r.Context()); err != nil {
		return err
	}

	httpio.RespondNoContent(w)
	return nil
}

type PrintWSTicketResponse struct {
	Ticket uuid.UUID `json:"ticket"`
}

func (h *AuthHandler) PrintWSTicket(w http.ResponseWriter, r *http.Request) error {
	ticket, err := h.service.PrintWSTicket(r.Context())
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, PrintWSTicketResponse{Ticket: ticket})
	return nil
}
