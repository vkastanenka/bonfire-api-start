package user

import (
	"net/http"

	"bonfire-api/internal/appctx"
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/presence"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// RegisterRoutes registers all user-related endpoints. Only this function is exported.
func RegisterRoutes(r chi.Router, service *Service, bind *httpio.Bind) {
	h := &handler{
		service: service,
		bind:    bind,
	}

	r.Get("/users/{id}", httpio.ToHTTPErr(h.get))

	r.Route("/users/@me", func(me chi.Router) {
		me.Patch("/email", httpio.ToHTTPErr(h.updateEmail))
		me.Patch("/username", httpio.ToHTTPErr(h.updateUsername))
		me.Post("/password", httpio.ToHTTPErr(h.updatePassword))
		me.Patch("/presence", httpio.ToHTTPErr(h.updatePreferredPresence))
		me.Patch("/profile", httpio.ToHTTPErr(h.updateProfile))
		me.Post("/disable", httpio.ToHTTPErr(h.disable))
		me.Delete("/", httpio.ToHTTPErr(h.scheduleDelete))
	})
}

type handler struct {
	service *Service
	bind    *httpio.Bind
}

type getPath struct {
	ID uuid.UUID `path:"id" validate:"required"`
}

func (h *handler) get(w http.ResponseWriter, r *http.Request) error {
	var path getPath
	if err := h.bind.Path(r, &path); err != nil {
		return err
	}

	u, err := h.service.Get(r.Context(), path.ID)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, NewDTO(u))
	return nil
}

type updateEmailRequest struct {
	Password string `json:"password" validate:"required,user_password"`
	NewEmail string `json:"newEmail" mod:"email" validate:"required,email_spec"`
}

func (h *handler) updateEmail(w http.ResponseWriter, r *http.Request) error {
	var req updateEmailRequest
	if err := h.bind.JSON(w, r, &req); err != nil {
		return err
	}

	claims, err := appctx.GetClaims(r.Context())
	if err != nil {
		return err
	}

	u, err := h.service.UpdateEmail(r.Context(), UpdateEmailParams{
		UserID:    claims.UserID,
		SessionID: claims.SessionID,
		NewEmail:  req.NewEmail,
		Password:  req.Password,
	})
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, NewMeDTO(u))
	return nil
}

type updateUsernameRequest struct {
	Password    string `json:"password" validate:"required,user_password"`
	NewUsername string `json:"newUsername" mod:"text" validate:"required,user_username"`
}

func (h *handler) updateUsername(w http.ResponseWriter, r *http.Request) error {
	var req updateUsernameRequest
	if err := h.bind.JSON(w, r, &req); err != nil {
		return err
	}

	claims, err := appctx.GetClaims(r.Context())
	if err != nil {
		return err
	}

	u, err := h.service.UpdateUsername(r.Context(), UpdateUsernameParams{
		UserID:      claims.UserID,
		SessionID:   claims.SessionID,
		NewUsername: req.NewUsername,
		Password:    req.Password,
	})
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, NewMeDTO(u))
	return nil
}

type updatePasswordRequest struct {
	CurrentPassword    string `json:"currentPassword" validate:"required,user_password"`
	NewPassword        string `json:"newPassword" validate:"required,user_password"`
	NewPasswordConfirm string `json:"newPasswordConfirm" validate:"required,eqfield=NewPassword"`
}

func (h *handler) updatePassword(w http.ResponseWriter, r *http.Request) error {
	var req updatePasswordRequest
	if err := h.bind.JSON(w, r, &req); err != nil {
		return err
	}

	claims, err := appctx.GetClaims(r.Context())
	if err != nil {
		return err
	}

	if err := h.service.UpdatePassword(r.Context(), UpdatePasswordParams{
		UserID:             claims.UserID,
		SessionID:          claims.SessionID,
		CurrentPassword:    req.CurrentPassword,
		NewPassword:        req.NewPassword,
		NewPasswordConfirm: req.NewPasswordConfirm,
	}); err != nil {
		return err
	}

	httpio.RespondNoContent(w)
	return nil
}

type updatePreferredPresenceRequest struct {
	Presence *presence.Presence         `json:"presence,omitempty" validate:"omitempty,user_preferred_presence"`
	Duration *PreferredPresenceDuration `json:"duration,omitempty" validate:"omitempty,user_preferred_presence_duration"`
}

func (h *handler) updatePreferredPresence(w http.ResponseWriter, r *http.Request) error {
	var req updatePreferredPresenceRequest
	if err := h.bind.JSON(w, r, &req); err != nil {
		return err
	}

	claims, err := appctx.GetClaims(r.Context())
	if err != nil {
		return err
	}

	u, err := h.service.UpdatePreferredPresence(r.Context(), UpdatePreferredPresenceParams{
		UserID:    claims.UserID,
		SessionID: claims.SessionID,
		Presence:  req.Presence,
		Duration:  req.Duration,
	})
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, NewMeDTO(u))
	return nil
}

type updateProfileRequest struct {
	DisplayName string  `json:"displayName" mod:"text" validate:"required,user_display_name"`
	Bio         *string `json:"bio,omitempty" mod:"text" validate:"omitempty,user_bio"`
	AvatarURL   *string `json:"avatarUrl,omitempty" validate:"omitempty,url_spec"`
	BannerColor *string `json:"bannerColor,omitempty" validate:"omitempty,hexcolor"`
}

func (h *handler) updateProfile(w http.ResponseWriter, r *http.Request) error {
	var req updateProfileRequest
	if err := h.bind.JSON(w, r, &req); err != nil {
		return err
	}

	claims, err := appctx.GetClaims(r.Context())
	if err != nil {
		return err
	}

	u, err := h.service.UpdateProfile(r.Context(), UpdateProfileParams{
		UserID:      claims.UserID,
		SessionID:   claims.SessionID,
		DisplayName: req.DisplayName,
		Bio:         req.Bio,
		AvatarURL:   req.AvatarURL,
		BannerColor: req.BannerColor,
	})
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, NewMeDTO(u))
	return nil
}

type disableRequest struct {
	Password string `json:"password" validate:"required,user_password"`
}

func (h *handler) disable(w http.ResponseWriter, r *http.Request) error {
	var req disableRequest
	if err := h.bind.JSON(w, r, &req); err != nil {
		return err
	}

	claims, err := appctx.GetClaims(r.Context())
	if err != nil {
		return err
	}

	if err := h.service.Disable(r.Context(), DisableParams{
		UserID:    claims.UserID,
		SessionID: claims.SessionID,
		Password:  req.Password,
	}); err != nil {
		return err
	}

	httpio.RespondNoContent(w)
	return nil
}

type scheduleDeleteRequest struct {
	Password string `json:"password" validate:"required,user_password"`
}

func (h *handler) scheduleDelete(w http.ResponseWriter, r *http.Request) error {
	var req scheduleDeleteRequest
	if err := h.bind.JSON(w, r, &req); err != nil {
		return err
	}

	claims, err := appctx.GetClaims(r.Context())
	if err != nil {
		return err
	}

	if err := h.service.ScheduleDelete(r.Context(), ScheduleDeleteParams{
		UserID:    claims.UserID,
		SessionID: claims.SessionID,
		Password:  req.Password,
	}); err != nil {
		return err
	}

	httpio.RespondNoContent(w)
	return nil
}
