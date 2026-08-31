package handler

import (
	"net/http"

	"bonfire-api/internal/channel"
	"bonfire-api/internal/fields"
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/relation"
	"bonfire-api/internal/user"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

type UserHandler struct {
	service     UserService
	relService  RelationService
	chanService ChannelService
	bind        *httpio.Bind
}

func NewUserHandler(service UserService, relService RelationService, chanService ChannelService, bind *httpio.Bind) *UserHandler {
	return &UserHandler{
		service:     service,
		relService:  relService,
		chanService: chanService,
		bind:        bind,
	}
}

type UserGetPath struct {
	UserID uuid.UUID `path:"userId" validate:"required,uuid"`
}

func (h *UserHandler) Get(w http.ResponseWriter, r *http.Request) error {
	var path UserGetPath
	if err := h.bind.Path(r, &path); err != nil {
		return err
	}

	u, err := h.service.GetView(r.Context(), path.UserID)
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, u)
	return nil
}

type UserGetMeResponse struct {
	Me             user.UserMeView                `json:"me"`
	Users          map[fields.ID]*user.UserView   `json:"users"`
	Presences      map[fields.ID]user.Presence    `json:"presences"`
	Channels       map[fields.ID]*channel.Channel `json:"channels"`
	Members        map[fields.ID]*channel.Member  `json:"members"`
	PeerIDs        map[fields.ID][]fields.ID      `json:"peerIDs"`
	FriendChannels map[fields.ID]fields.ID        `json:"friendChannels"` // friendID -> channelID
	ChannelIDs     []fields.ID                    `json:"channelIds"`     // ordered sidebar channels
	FriendIDs      []fields.ID                    `json:"friendIds"`      // sorted friend IDs
}

func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	userID, err := httpio.CtxGetUserID(ctx)
	if err != nil {
		return err
	}

	g1, gCtx1 := errgroup.WithContext(ctx)

	var (
		me                *user.User
		channelMap        map[fields.ID]*channel.Channel
		memberMap         map[fields.ID]*channel.Member
		peerIDsMap        map[fields.ID][]fields.ID
		channelIDs        []fields.ID
		peerIDs           []fields.ID
		friendChannelsMap map[fields.ID]fields.ID
		friendIDs         []fields.ID
	)

	g1.Go(func() error {
		var err error
		me, err = h.service.Get(gCtx1, userID.UUID())
		return err
	})

	g1.Go(func() error {
		var err error
		channelMap, memberMap, peerIDsMap, channelIDs, peerIDs, _, err = h.chanService.GetSidebar(gCtx1, userID.UUID())
		return err
	})

	g1.Go(func() error {
		var err error
		friendChannelsMap, friendIDs, err = h.relService.GetPeers(gCtx1, userID.UUID(), relation.NewTypeFriends().String())
		return err
	})

	if err := g1.Wait(); err != nil {
		return err
	}

	// Phase 2: Deduplicate IDs & fetch user models + presences concurrently
	allUserIDs := append(peerIDs, friendIDs...)
	dedupedUserIDs := fields.DedupeIDs(allUserIDs)

	g2, gCtx2 := errgroup.WithContext(ctx)

	var (
		usersMap     map[fields.ID]*user.User
		presencesMap map[fields.ID]user.Presence
	)

	g2.Go(func() error {
		var err error
		usersMap, err = h.service.GetBatch(gCtx2, dedupedUserIDs)
		return err
	})

	g2.Go(func() error {
		var err error
		presencesMap, err = h.service.GetBatchPresence(gCtx2, dedupedUserIDs)
		return err
	})

	if err := g2.Wait(); err != nil {
		return err
	}

	// Phase 3: Sort friends list by display name/username
	relation.SortFriendIDs(friendIDs, usersMap)

	// Phase 4: Map domain users to user views (stripping internal fields like password hashes)
	userViews := make(map[fields.ID]*user.UserView, len(usersMap))
	for id, u := range usersMap {
		if u == nil {
			continue
		}
		p := presencesMap[id]
		view := user.ToUserView(u, p, fields.Now())
		userViews[id] = &view
	}

	// Phase 5: Construct normalized response
	response := UserGetMeResponse{
		Me:             user.ToUserMeView(me),
		Users:          userViews,
		Presences:      presencesMap,
		Channels:       channelMap,
		Members:        memberMap,
		PeerIDs:        peerIDsMap,
		FriendChannels: friendChannelsMap,
		ChannelIDs:     channelIDs,
		FriendIDs:      friendIDs,
	}

	httpio.RespondOK(w, r, response)
	return nil
}

// func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) error {
// 	ctx := r.Context()
// 	userID, err := httpio.CtxGetUserID(ctx)
// 	if err != nil {
// 		return err
// 	}

// 	g, gCtx := errgroup.WithContext(ctx)

// 	var (
// 		me       *user.User
// 		friends  []relation.Peer
// 		channels []channel.SidebarView
// 	)

// 	g.Go(func() error {
// 		var err error
// 		me, err = h.service.Get(gCtx, userID.UUID())
// 		return err
// 	})

// 	g.Go(func() error {
// 		var err error
// 		friends, err = h.relService.GetPeers(gCtx, userID.UUID(), relation.NewTypeFriends().String())
// 		return err
// 	})

// 	g.Go(func() error {
// 		var err error
// 		channels, err = h.chanService.GetSidebar(gCtx, userID.UUID())
// 		return err
// 	})

// 	if err := g.Wait(); err != nil {
// 		return err
// 	}

// 	httpio.RespondOK(w, r, UserGetMeResponse{
// 		Me:       user.ToUserMeView(me),
// 		Friends:  friends,
// 		Channels: channels,
// 	})
// 	return nil
// }

type UserUpdateEmailRequest struct {
	NewEmail string `json:"newEmail" mod:"email" validate:"required,email,max=255"`
	Password string `json:"password" validate:"required,min=12,max=255"`
}

func (h *UserHandler) UpdateEmail(w http.ResponseWriter, r *http.Request) error {
	userID, err := httpio.CtxGetUserID(r.Context())
	if err != nil {
		return err
	}

	var req UserUpdateEmailRequest
	if err := h.bind.JSON(w, r, &req); err != nil {
		return err
	}

	u, err := h.service.UpdateEmail(r.Context(), user.UpdateEmailParams{
		UserID:   userID.UUID(),
		NewEmail: req.NewEmail,
		Password: req.Password,
	})
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, user.ToUserMeView(u))
	return nil
}

type UserUpdateUsernameRequest struct {
	NewUsername string `json:"newUsername" mod:"text" validate:"required,alphanum,min=3,max=32"`
	Password    string `json:"password" validate:"required,min=12,max=255"`
}

func (h *UserHandler) UpdateUsername(w http.ResponseWriter, r *http.Request) error {
	userID, err := httpio.CtxGetUserID(r.Context())
	if err != nil {
		return err
	}

	var req UserUpdateUsernameRequest
	if err := h.bind.JSON(w, r, &req); err != nil {
		return err
	}

	u, err := h.service.UpdateUsername(r.Context(), user.UpdateUsernameParams{
		UserID:      userID.UUID(),
		NewUsername: req.NewUsername,
		Password:    req.Password,
	})
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, user.ToUserMeView(u))
	return nil
}

type UserUpdatePasswordRequest struct {
	CurrentPassword    string `json:"currentPassword" validate:"required,min=12,max=255"`
	NewPassword        string `json:"newPassword" validate:"required,min=12,max=255"`
	NewPasswordConfirm string `json:"newPasswordConfirm" validate:"required,min=12,max=255"`
}

func (h *UserHandler) UpdatePassword(w http.ResponseWriter, r *http.Request) error {
	userID, err := httpio.CtxGetUserID(r.Context())
	if err != nil {
		return err
	}

	var req UserUpdatePasswordRequest
	if err := h.bind.JSON(w, r, &req); err != nil {
		return err
	}

	if err := h.service.UpdatePassword(r.Context(), user.UpdatePasswordParams{
		UserID:             userID.UUID(),
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
	Presence *string `json:"presence,omitempty"`
	Duration *string `json:"duration,omitempty"`
}

func (h *UserHandler) UpdatePreferredPresence(w http.ResponseWriter, r *http.Request) error {
	userID, err := httpio.CtxGetUserID(r.Context())
	if err != nil {
		return err
	}

	var req UserUpdatePreferredPresenceRequest
	if err := h.bind.JSON(w, r, &req); err != nil {
		return err
	}

	u, err := h.service.UpdatePreferredPresence(r.Context(), user.UpdatePreferredPresenceParams{
		UserID:   userID.UUID(),
		Presence: req.Presence,
		Duration: req.Duration,
	})
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, user.ToUserMeView(u))
	return nil
}

type UserUpdateProfileRequest struct {
	DisplayName string  `json:"displayName" mod:"text" validate:"required,min=3,max=32"`
	Bio         *string `json:"bio,omitempty" mod:"text" validate:"omitempty,max=190"`
	AvatarURL   *string `json:"avatarUrl,omitempty" validate:"omitempty,url,max=2048"`
	BannerColor *string `json:"bannerColor,omitempty" validate:"omitempty,hexcolor"`
}

func (h *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) error {
	userID, err := httpio.CtxGetUserID(r.Context())
	if err != nil {
		return err
	}

	var req UserUpdateProfileRequest
	if err := h.bind.JSON(w, r, &req); err != nil {
		return err
	}

	u, err := h.service.UpdateProfile(r.Context(), user.UpdateProfileParams{
		UserID:      userID.UUID(),
		DisplayName: req.DisplayName,
		Bio:         req.Bio,
		AvatarURL:   req.AvatarURL,
		BannerColor: req.BannerColor,
	})
	if err != nil {
		return err
	}

	httpio.RespondOK(w, r, user.ToUserMeView(u))
	return nil
}

type UserDisableRequest struct {
	Password string `json:"password" validate:"required,min=12,max=255"`
}

func (h *UserHandler) Disable(w http.ResponseWriter, r *http.Request) error {
	userID, err := httpio.CtxGetUserID(r.Context())
	if err != nil {
		return err
	}

	var req UserDisableRequest
	if err := h.bind.JSON(w, r, &req); err != nil {
		return err
	}

	if err := h.service.Disable(r.Context(), user.DisableParams{
		UserID:   userID.UUID(),
		Password: req.Password,
	}); err != nil {
		return err
	}

	httpio.RespondNoContent(w)
	return nil
}

type UserScheduleDeleteRequest struct {
	Password string `json:"password" validate:"required,min=12,max=255"`
}

func (h *UserHandler) ScheduleDelete(w http.ResponseWriter, r *http.Request) error {
	userID, err := httpio.CtxGetUserID(r.Context())
	if err != nil {
		return err
	}

	var req UserScheduleDeleteRequest
	if err := h.bind.JSON(w, r, &req); err != nil {
		return err
	}

	if err := h.service.ScheduleDelete(r.Context(), user.ScheduleDeleteParams{
		UserID:   userID.UUID(),
		Password: req.Password,
	}); err != nil {
		return err
	}

	httpio.RespondNoContent(w)
	return nil
}
