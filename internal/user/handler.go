package user

import (
	"net/http"

	"bonfire-api/internal/appctx"
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/presence"

	"github.com/google/uuid"
)

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

type GetPath struct {
	ID uuid.UUID `path:"id" validate:"required"`
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) error {
	var path GetPath
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

type UpdateEmailRequest struct {
	Password string `json:"password" validate:"required,user_password"`
	NewEmail string `json:"newEmail" mod:"email" validate:"required,email_spec"`
}

func (h *Handler) UpdateEmail(w http.ResponseWriter, r *http.Request) error {
	var req UpdateEmailRequest
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

type UpdateUsernameRequest struct {
	Password    string `json:"password" validate:"required,user_password"`
	NewUsername string `json:"newUsername" mod:"text" validate:"required,user_username"`
}

func (h *Handler) UpdateUsername(w http.ResponseWriter, r *http.Request) error {
	var req UpdateUsernameRequest
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

type UpdatePasswordRequest struct {
	CurrentPassword    string `json:"currentPassword" validate:"required,user_password"`
	NewPassword        string `json:"newPassword" validate:"required,user_password"`
	NewPasswordConfirm string `json:"newPasswordConfirm" validate:"required,eqfield=NewPassword"`
}

func (h *Handler) UpdatePassword(w http.ResponseWriter, r *http.Request) error {
	var req UpdatePasswordRequest
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

type UpdatePreferredPresenceRequest struct {
	Presence *presence.Presence         `json:"presence,omitempty" validate:"omitempty,user_preferred_presence"`
	Duration *PreferredPresenceDuration `json:"duration,omitempty" validate:"omitempty,user_preferred_presence_duration"`
}

func (h *Handler) UpdatePreferredPresence(w http.ResponseWriter, r *http.Request) error {
	var req UpdatePreferredPresenceRequest
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

type UpdateProfileRequest struct {
	DisplayName string  `json:"displayName" mod:"text" validate:"required,user_display_name"`
	Bio         *string `json:"bio,omitempty" mod:"text" validate:"omitempty,user_bio"`
	AvatarURL   *string `json:"avatarUrl,omitempty" validate:"omitempty,url_spec"`
	BannerColor *string `json:"bannerColor,omitempty" validate:"omitempty,hexcolor"`
}

func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) error {
	var req UpdateProfileRequest
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

type DisableRequest struct {
	Password string `json:"password" validate:"required,user_password"`
}

func (h *Handler) Disable(w http.ResponseWriter, r *http.Request) error {
	var req DisableRequest
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

type ScheduleDeleteRequest struct {
	Password string `json:"password" validate:"required,user_password"`
}

func (h *Handler) ScheduleDelete(w http.ResponseWriter, r *http.Request) error {
	var req ScheduleDeleteRequest
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
