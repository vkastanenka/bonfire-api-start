package handler

import (
	"context"
	"net/http"

	"bonfire-api/internal/httpio"
	"bonfire-api/internal/presence"
	"bonfire-api/internal/relationship"
	"bonfire-api/internal/user"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

type MeRelationshipService interface {
	ListPerspectives(ctx context.Context, userID uuid.UUID, filter *relationship.Variant) ([]relationship.Perspective, error)
	GetPerspective(ctx context.Context, userID, peerID uuid.UUID) (*relationship.Perspective, error)
	SendFriendRequest(ctx context.Context, actorID, targetID uuid.UUID) error
	AcceptFriendRequest(ctx context.Context, actorID, peerID uuid.UUID) error
	Block(ctx context.Context, actorID, peerID uuid.UUID) error
	DeleteVerified(ctx context.Context, actorID, peerID uuid.UUID) error
}

type MeUserService interface {
	Get(ctx context.Context, id uuid.UUID) (*user.User, error)
	UpdateEmail(ctx context.Context, p user.UpdateEmailParams) (*user.User, error)
	UpdateUsername(ctx context.Context, p user.UpdateUsernameParams) (*user.User, error)
	UpdatePassword(ctx context.Context, p user.UpdatePasswordParams) error
	UpdatePreferredPresence(ctx context.Context, p user.UpdatePreferredPresenceParams) (*user.User, error)
	UpdateProfile(ctx context.Context, p user.UpdateProfileParams) (*user.User, error)
	Disable(ctx context.Context, p user.DisableParams) error
	ScheduleDelete(ctx context.Context, p user.ScheduleDeleteParams) error
}

type MePresenceService interface {
	GetBulk(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]presence.Presence, error)
}

type Me struct {
	relationships MeRelationshipService
	users         MeUserService
	presence      MePresenceService
	// channels      channel.Service
	bind *httpio.Bind
}

func NewMe(
	r MeRelationshipService,
	u MeUserService,
	p MePresenceService,
	// c channel.Service,
	bind *httpio.Bind,
) *Me {
	return &Me{
		relationships: r,
		users:         u,
		presence:      p,
		// channels:      c,
		bind: bind,
	}
}

type MeGetResponse struct {
	Me            MeResponse                      `json:"me"`
	Relationships []RelationshipResponse          `json:"relationships"`
	Presences     map[uuid.UUID]presence.Presence `json:"presences"`
	// Channels      []channel.PrivateChannelResponse `json:"channels"`
}

func (h *Me) Get(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	userID, err := httpio.CtxGetUserID(ctx)
	if err != nil {
		return err
	}

	g, gCtx := errgroup.WithContext(ctx)

	var (
		meUser *user.User
		// relPersp []relationship.Perspective
		// privChannels []channel.PrivateChannel
	)

	g.Go(func() error {
		var err error
		meUser, err = h.users.Get(gCtx, userID)
		return err
	})

	// g.Go(func() error {
	// 	var err error
	// 	relPersp, err = h.relationships.ListPerspectives(gCtx, userID, nil)
	// 	return err
	// })

	// if err := g.Wait(); err != nil {
	// 	return err
	// }

	// peerIDs := make([]uuid.UUID, 0, len(relPersp)+1)
	// peerIDs = append(peerIDs, userID)
	// for _, p := range relPersp {
	// 	peerIDs = append(peerIDs, p.PeerID())
	// }

	// presences, err := h.presence.GetBulk(ctx, peerIDs)
	// if err != nil {
	// 	return err
	// }

	// relResponses := make([]RelationshipResponse, len(relPersp))
	// for i, p := range relPersp {
	// 	var peerPresence *presence.Presence
	// 	if ps, ok := presences[p.PeerID()]; ok {
	// 		peerPresence = &ps
	// 	}
	// 	relResponses[i] = ToRelationshipResponse(p, peerPresence)
	// }

	httpio.RespondOK(w, r, MeGetResponse{
		Me: ToMeResponse(*meUser),
		// Relationships: relResponses,
		// Presences:     presences,
	})
	return nil
}

type UpdateEmailRequest struct {
	NewEmail string `json:"newEmail" mod:"email" validate:"required,email,max=255"`
	Password string `json:"password" validate:"required,min=12,max=255"`
}

func (h *Me) UpdateEmail(w http.ResponseWriter, r *http.Request) error {
	userID, err := httpio.CtxGetUserID(r.Context())
	if err != nil {
		return err
	}

	var req UpdateEmailRequest
	if err := h.bind.JSON(w, r, &req); err != nil {
		return err
	}

	updatedUser, err := h.users.UpdateEmail(r.Context(), user.UpdateEmailParams{
		UserID:   userID,
		NewEmail: req.NewEmail,
		Password: req.Password,
	})
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, ToUserResponse(*updatedUser))
	return nil
}

type UpdateUsernameRequest struct {
	NewUsername string `json:"newUsername" mod:"text" validate:"required,alphanum,min=3,max=32"`
	Password    string `json:"password" validate:"required,min=12,max=255"`
}

func (h *Me) UpdateUsername(w http.ResponseWriter, r *http.Request) error {
	userID, err := httpio.CtxGetUserID(r.Context())
	if err != nil {
		return err
	}

	var req UpdateUsernameRequest
	if err := h.bind.JSON(w, r, &req); err != nil {
		return err
	}

	updatedUser, err := h.users.UpdateUsername(r.Context(), user.UpdateUsernameParams{
		UserID:      userID,
		NewUsername: req.NewUsername,
		Password:    req.Password,
	})
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, ToUserResponse(*updatedUser))
	return nil
}

type UpdatePasswordRequest struct {
	CurrentPassword    string `json:"currentPassword" validate:"required,min=12,max=255"`
	NewPassword        string `json:"newPassword" validate:"required,min=12,max=255"`
	NewPasswordConfirm string `json:"newPasswordConfirm" validate:"required,min=12,max=255"`
}

func (h *Me) UpdatePassword(w http.ResponseWriter, r *http.Request) error {
	userID, err := httpio.CtxGetUserID(r.Context())
	if err != nil {
		return err
	}

	var req UpdatePasswordRequest
	if err := h.bind.JSON(w, r, &req); err != nil {
		return err
	}

	err = h.users.UpdatePassword(r.Context(), user.UpdatePasswordParams{
		UserID:             userID,
		CurrentPassword:    req.CurrentPassword,
		NewPassword:        req.NewPassword,
		NewPasswordConfirm: req.NewPasswordConfirm,
	})
	if err != nil {
		return err
	}

	httpio.RespondNoContent(w)
	return nil
}

type UpdatePreferredPresenceRequest struct {
	Presence *string `json:"presence,omitempty" validate:"omitempty,oneof=online idle dnd offline"`
	Until    *string `json:"until,omitempty" validate:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`
}

func (h *Me) UpdatePreferredPresence(w http.ResponseWriter, r *http.Request) error {
	userID, err := httpio.CtxGetUserID(r.Context())
	if err != nil {
		return err
	}

	var req UpdatePreferredPresenceRequest
	if err := h.bind.JSON(w, r, &req); err != nil {
		return err
	}

	updatedUser, err := h.users.UpdatePreferredPresence(r.Context(), user.UpdatePreferredPresenceParams{
		UserID:   userID,
		Presence: req.Presence,
		Until:    req.Until,
	})
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, ToUserResponse(*updatedUser))
	return nil
}

type UpdateProfileRequest struct {
	DisplayName string  `json:"displayName" mod:"text" validate:"required,min=3,max=32"`
	Bio         *string `json:"bio,omitempty" mod:"text" validate:"omitempty,max=190"`
	AvatarURL   *string `json:"avatarUrl,omitempty" validate:"omitempty,url"`
	BannerColor *string `json:"bannerColor,omitempty" validate:"omitempty,hexcolor"`
}

func (h *Me) UpdateProfile(w http.ResponseWriter, r *http.Request) error {
	userID, err := httpio.CtxGetUserID(r.Context())
	if err != nil {
		return err
	}

	var req UpdateProfileRequest
	if err := h.bind.JSON(w, r, &req); err != nil {
		return err
	}

	updatedUser, err := h.users.UpdateProfile(r.Context(), user.UpdateProfileParams{
		UserID:      userID,
		DisplayName: req.DisplayName,
		Bio:         req.Bio,
		AvatarURL:   req.AvatarURL,
		BannerColor: req.BannerColor,
	})
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, ToUserResponse(*updatedUser))
	return nil
}

type AccountActionWithPasswordRequest struct {
	Password string `json:"password" validate:"required,min=12,max=255"`
}

func (h *Me) Disable(w http.ResponseWriter, r *http.Request) error {
	userID, err := httpio.CtxGetUserID(r.Context())
	if err != nil {
		return err
	}

	var req AccountActionWithPasswordRequest
	if err := h.bind.JSON(w, r, &req); err != nil {
		return err
	}

	if err := h.users.Disable(r.Context(), user.DisableParams{
		UserID:   userID,
		Password: req.Password,
	}); err != nil {
		return err
	}

	httpio.RespondNoContent(w)
	return nil
}

func (h *Me) ScheduleDelete(w http.ResponseWriter, r *http.Request) error {
	userID, err := httpio.CtxGetUserID(r.Context())
	if err != nil {
		return err
	}

	var req AccountActionWithPasswordRequest
	if err := h.bind.JSON(w, r, &req); err != nil {
		return err
	}

	if err := h.users.ScheduleDelete(r.Context(), user.ScheduleDeleteParams{
		UserID:   userID,
		Password: req.Password,
	}); err != nil {
		return err
	}

	httpio.RespondNoContent(w)
	return nil
}

type MeSendFriendRequestPath struct {
	ID uuid.UUID `path:"id" validate:"required,uuid"`
}

func (h *Me) SendFriendRequest(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	actorID, err := httpio.CtxGetUserID(ctx)
	if err != nil {
		return err
	}

	var path MeSendFriendRequestPath
	if err := h.bind.Path(r, &path); err != nil {
		return err
	}

	if err := h.relationships.SendFriendRequest(ctx, actorID, path.ID); err != nil {
		return err
	}

	h.respondWithPerspective(w, r, actorID, path.ID)
	return nil
}

type MeAcceptFriendRequestPath struct {
	ID uuid.UUID `path:"id" validate:"required,uuid"`
}

func (h *Me) AcceptFriendRequest(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	actorID, err := httpio.CtxGetUserID(ctx)
	if err != nil {
		return err
	}

	var path MeAcceptFriendRequestPath
	if err := h.bind.Path(r, &path); err != nil {
		return err
	}

	if err := h.relationships.AcceptFriendRequest(ctx, actorID, path.ID); err != nil {
		return err
	}

	h.respondWithPerspective(w, r, actorID, path.ID)
	return nil
}

type MeBlockPath struct {
	ID uuid.UUID `path:"id" validate:"required,uuid"`
}

func (h *Me) Block(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	actorID, err := httpio.CtxGetUserID(ctx)
	if err != nil {
		return err
	}

	var path MeBlockPath
	if err := h.bind.Path(r, &path); err != nil {
		return err
	}

	if err := h.relationships.Block(ctx, actorID, path.ID); err != nil {
		return err
	}

	h.respondWithPerspective(w, r, actorID, path.ID)
	return nil
}

type MeRemoveRelationshipPath struct {
	ID uuid.UUID `path:"id" validate:"required,uuid"`
}

func (h *Me) RemoveRelationship(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	actorID, err := httpio.CtxGetUserID(ctx)
	if err != nil {
		return err
	}

	var path MeRemoveRelationshipPath
	if err := h.bind.Path(r, &path); err != nil {
		return err
	}

	if err := h.relationships.DeleteVerified(ctx, actorID, path.ID); err != nil {
		return err
	}

	httpio.RespondNoContent(w)
	return nil
}

func (h *Me) respondWithPerspective(
	w http.ResponseWriter,
	r *http.Request,
	userID, peerID uuid.UUID,
) error {
	ctx := r.Context()

	perspective, err := h.relationships.GetPerspective(ctx, userID, peerID)
	if err != nil {
		return err
	}

	var peerPresence *presence.Presence
	presences, err := h.presence.GetBulk(ctx, []uuid.UUID{peerID})
	if err == nil {
		if ps, ok := presences[peerID]; ok {
			peerPresence = &ps
		}
	}

	httpio.RespondOK(w, r, ToRelationshipResponse(*perspective, peerPresence))
	return nil
}
