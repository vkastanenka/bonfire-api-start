package auth

import (
	"net/http"

	"bonfire-api/internal/appctx"
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/pkg/errs"
	"bonfire-api/internal/user"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type accessTokenResponse struct {
	AccessToken string `json:"accessToken"`
}

type Handler struct {
	service *Service
	bind    *httpio.Bind
}

func NewHandler(service *Service, bind *httpio.Bind) *Handler {
	return &Handler{
		service: service,
		bind:    bind,
	}
}

func (h *Handler) RegisterPublicRoutes(r chi.Router) {
	r.Post("/auth/register", httpio.ToHTTPErr(h.register))
	r.Post("/auth/login", httpio.ToHTTPErr(h.login))
	r.Post("/auth/refresh", httpio.ToHTTPErr(h.refresh))
	r.Post("/auth/verify", httpio.ToHTTPErr(h.verifyEmail))
	r.Post("/auth/forgot-password", httpio.ToHTTPErr(h.forgotPassword))
	r.Post("/auth/reset-password", httpio.ToHTTPErr(h.resetPassword))
}

func (h *Handler) RegisterAuthenticatedRoutes(r chi.Router) {
	r.Post("/auth/resend-verify", httpio.ToHTTPErr(h.resendVerify))

	r.Route("/users/@me/sessions", func(sessions chi.Router) {
		sessions.Get("/", httpio.ToHTTPErr(h.listValidSessions))
		sessions.Delete("/", httpio.ToHTTPErr(h.revokeAllSessions))
		sessions.Delete("/{id}", httpio.ToHTTPErr(h.revokeSession))
	})
}

type registerRequest struct {
	Email       string  `json:"email" mod:"email" validate:"required,email_spec"`
	Username    string  `json:"username" mod:"text" validate:"required,user_username"`
	DisplayName *string `json:"displayName,omitempty" mod:"text" validate:"omitempty,user_display_name"`
	Password    string  `json:"password" validate:"required,user_password"`
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) error {
	clientMeta, err := appctx.GetClientMeta(r.Context())
	if err != nil {
		return err
	}

	var req registerRequest
	if err = h.bind.JSON(w, r, &req); err != nil {
		return err
	}

	data, err := h.service.Register(r.Context(), RegisterParams{
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
	httpio.RespondCreated(w, r, accessTokenResponse{AccessToken: data.AccessToken})
	return nil
}

type loginRequest struct {
	Email    string `json:"email" mod:"email" validate:"required,email_spec"`
	Password string `json:"password" validate:"required,user_password"`
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) error {
	clientMeta, err := appctx.GetClientMeta(r.Context())
	if err != nil {
		return err
	}

	var req loginRequest
	if err = h.bind.JSON(w, r, &req); err != nil {
		return err
	}

	data, err := h.service.Login(r.Context(), LoginParams{
		Email:      req.Email,
		Password:   req.Password,
		ClientMeta: clientMeta,
	})
	if err != nil {
		return err
	}

	httpio.CookieSetRefreshToken(w, data.RefreshToken, data.RefreshTokenExpiresAt)
	httpio.RespondOK(w, r, accessTokenResponse{AccessToken: data.AccessToken})
	return nil
}

type forgotPasswordRequest struct {
	Email string `json:"email" mod:"email" validate:"required,email_spec"`
}

func (h *Handler) forgotPassword(w http.ResponseWriter, r *http.Request) error {
	var req forgotPasswordRequest
	if err := h.bind.JSON(w, r, &req); err != nil {
		return err
	}

	if err := h.service.ForgotPassword(r.Context(), req.Email); err != nil {
		return err
	}

	httpio.RespondNoContent(w)
	return nil
}

type resetPasswordRequest struct {
	Token    string `json:"token" validate:"required,token"`
	Password string `json:"password" validate:"required,user_password"`
}

func (h *Handler) resetPassword(w http.ResponseWriter, r *http.Request) error {
	clientMeta, err := appctx.GetClientMeta(r.Context())
	if err != nil {
		return err
	}

	var req resetPasswordRequest
	if err = h.bind.JSON(w, r, &req); err != nil {
		return err
	}

	data, err := h.service.ResetPassword(r.Context(), ResetPasswordParams{
		Token:      req.Token,
		Password:   req.Password,
		ClientMeta: clientMeta,
	})
	if err != nil {
		return err
	}

	httpio.CookieSetRefreshToken(w, data.RefreshToken, data.RefreshTokenExpiresAt)
	httpio.RespondOK(w, r, accessTokenResponse{AccessToken: data.AccessToken})
	return nil
}

func (h *Handler) refresh(w http.ResponseWriter, r *http.Request) error {
	refreshToken, err := httpio.CookieGetRefreshToken(r)
	if err != nil {
		return errs.Unauthenticated("Missing refresh token, please log in.").Wrap(err)
	}

	clientMeta, err := appctx.GetClientMeta(r.Context())
	if err != nil {
		return err
	}

	data, err := h.service.Refresh(r.Context(), RefreshParams{
		RefreshToken: refreshToken,
		ClientMeta:   clientMeta,
	})
	if err != nil {
		return err
	}

	httpio.CookieSetRefreshToken(w, data.RefreshToken, data.RefreshTokenExpiresAt)
	httpio.RespondOK(w, r, accessTokenResponse{AccessToken: data.AccessToken})
	return nil
}

type verifyEmailRequest struct {
	Token string `json:"token" validate:"required,token"`
}

func (h *Handler) verifyEmail(w http.ResponseWriter, r *http.Request) error {
	var req verifyEmailRequest
	if err := h.bind.JSON(w, r, &req); err != nil {
		return err
	}

	u, err := h.service.VerifyEmail(r.Context(), req.Token)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, user.NewMeDTO(u))
	return nil
}

func (h *Handler) resendVerify(w http.ResponseWriter, r *http.Request) error {
	claims, err := appctx.GetClaims(r.Context())
	if err != nil {
		return err
	}

	if err := h.service.ResendVerify(r.Context(), ResendVerifyParams{
		UserID:    claims.UserID,
		SessionID: claims.SessionID,
	}); err != nil {
		return err
	}

	httpio.RespondNoContent(w)
	return nil
}

func (h *Handler) listValidSessions(w http.ResponseWriter, r *http.Request) error {
	claims, err := appctx.GetClaims(r.Context())
	if err != nil {
		return err
	}

	sessions, err := h.service.ListValidSessions(r.Context(), claims.UserID)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, NewSessionDTOs(sessions))
	return nil
}

type sessionPath struct {
	ID uuid.UUID `path:"id" validate:"required"`
}

func (h *Handler) revokeSession(w http.ResponseWriter, r *http.Request) error {
	claims, err := appctx.GetClaims(r.Context())
	if err != nil {
		return err
	}

	var path sessionPath
	if err := h.bind.Path(r, &path); err != nil {
		return err
	}

	if err := h.service.RevokeSession(r.Context(), RevokeSessionParams{
		UserID:          claims.UserID,
		SessionID:       claims.SessionID,
		RevokeSessionID: path.ID,
	}); err != nil {
		return err
	}

	httpio.RespondNoContent(w)
	return nil
}

func (h *Handler) revokeAllSessions(w http.ResponseWriter, r *http.Request) error {
	claims, err := appctx.GetClaims(r.Context())
	if err != nil {
		return err
	}

	if err := h.service.RevokeAllSessions(r.Context(), RevokeAllSessionsParams{
		UserID:    claims.UserID,
		SessionID: claims.SessionID,
	}); err != nil {
		return err
	}

	httpio.RespondNoContent(w)
	return nil
}
