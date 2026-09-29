package handler

import (
	"net/http"

	"bonfire-api/internal/httpio"
	"bonfire-api/internal/presence"
	"bonfire-api/internal/user"

	"github.com/google/uuid"
)

type UserHandler struct {
	service UserService
	bind    *httpio.Bind
}

func NewUserHandler(service UserService, bind *httpio.Bind) *UserHandler {
	return &UserHandler{
		service: service,
		bind:    bind,
	}
}

type UserGetPath struct {
	UserID uuid.UUID `path:"userID" validate:"required"`
}

func (h *UserHandler) Get(w http.ResponseWriter, r *http.Request) error {
	var path UserGetPath
	if err := h.bind.Path(r, &path); err != nil {
		return err
	}

	u, err := h.service.Get(r.Context(), path.UserID)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, user.ParseView(u))
	return nil
}

type UserUpdateEmailRequest struct {
	NewEmail string `json:"newEmail" mod:"email" validate:"required,email_spec"`
	Password string `json:"password" validate:"required,user_password"`
}

func (h *UserHandler) UpdateEmail(w http.ResponseWriter, r *http.Request) error {
	var req UserUpdateEmailRequest
	if err := h.bind.JSON(w, r, &req); err != nil {
		return err
	}

	u, err := h.service.UpdateEmail(r.Context(), user.UpdateEmailParams{
		NewEmail: req.NewEmail,
		Password: req.Password,
	})
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, user.ParseMe(u))
	return nil
}

type UserUpdateUsernameRequest struct {
	NewUsername string `json:"newUsername" mod:"text" validate:"required,user_username"`
	Password    string `json:"password" validate:"required,user_password"`
}

func (h *UserHandler) UpdateUsername(w http.ResponseWriter, r *http.Request) error {
	var req UserUpdateUsernameRequest
	if err := h.bind.JSON(w, r, &req); err != nil {
		return err
	}

	u, err := h.service.UpdateUsername(r.Context(), user.UpdateUsernameParams{
		NewUsername: req.NewUsername,
		Password:    req.Password,
	})
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, user.ParseMe(u))
	return nil
}

type UserUpdatePasswordRequest struct {
	CurrentPassword    string `json:"currentPassword" validate:"required,user_password"`
	NewPassword        string `json:"newPassword" validate:"required,user_password"`
	NewPasswordConfirm string `json:"newPasswordConfirm" validate:"required,eqfield=NewPassword"`
}

func (h *UserHandler) UpdatePassword(w http.ResponseWriter, r *http.Request) error {
	var req UserUpdatePasswordRequest
	if err := h.bind.JSON(w, r, &req); err != nil {
		return err
	}

	if err := h.service.UpdatePassword(r.Context(), user.UpdatePasswordParams{
		CurrentPassword:    req.CurrentPassword,
		NewPassword:        req.NewPassword,
		NewPasswordConfirm: req.NewPasswordConfirm,
	}); err != nil {
		return err
	}

	httpio.RespondNoContent(w)
	return nil
}

type UserUpdatePreferredPresenceRequest struct {
	Presence *presence.Presence              `json:"presence,omitempty" validate:"omitempty,oneof=4 5 6"`
	Duration *user.PreferredPresenceDuration `json:"duration,omitempty" validate:"omitempty,oneof=1 2 3 4 5 6"`
}

func (h *UserHandler) UpdatePreferredPresence(w http.ResponseWriter, r *http.Request) error {
	var req UserUpdatePreferredPresenceRequest
	if err := h.bind.JSON(w, r, &req); err != nil {
		return err
	}

	u, err := h.service.UpdatePreferredPresence(r.Context(), user.UpdatePreferredPresenceParams{
		Presence: req.Presence,
		Duration: req.Duration,
	})
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, user.ParseMe(u))
	return nil
}

type UserUpdateProfileRequest struct {
	DisplayName string  `json:"displayName" mod:"text" validate:"required,user_display_name"`
	Bio         *string `json:"bio,omitempty" mod:"text" validate:"omitempty,user_bio"`
	AvatarURL   *string `json:"avatarUrl,omitempty" validate:"omitempty,url_spec"`
	BannerColor *string `json:"bannerColor,omitempty" validate:"omitempty,hexcolor"`
}

func (h *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) error {
	var req UserUpdateProfileRequest
	if err := h.bind.JSON(w, r, &req); err != nil {
		return err
	}

	u, err := h.service.UpdateProfile(r.Context(), user.UpdateProfileParams{
		DisplayName: req.DisplayName,
		Bio:         req.Bio,
		AvatarURL:   req.AvatarURL,
		BannerColor: req.BannerColor,
	})
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, user.ParseMe(u))
	return nil
}

type UserDisableRequest struct {
	Password string `json:"password" validate:"required,user_password"`
}

func (h *UserHandler) Disable(w http.ResponseWriter, r *http.Request) error {
	var req UserDisableRequest
	if err := h.bind.JSON(w, r, &req); err != nil {
		return err
	}

	if err := h.service.Disable(r.Context(), user.DisableParams{
		Password: req.Password,
	}); err != nil {
		return err
	}

	httpio.RespondNoContent(w)
	return nil
}

type UserScheduleDeleteRequest struct {
	Password string `json:"password" validate:"required,user_password"`
}

func (h *UserHandler) ScheduleDelete(w http.ResponseWriter, r *http.Request) error {
	var req UserScheduleDeleteRequest
	if err := h.bind.JSON(w, r, &req); err != nil {
		return err
	}

	if err := h.service.ScheduleDelete(r.Context(), user.ScheduleDeleteParams{
		Password: req.Password,
	}); err != nil {
		return err
	}

	httpio.RespondNoContent(w)
	return nil
}
