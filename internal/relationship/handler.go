package relationship

// // --- relationship handler ---

// type Handler struct {
// 	service *Service
// }

// func NewHandler(service *Service) *Handler {
// 	return &Handler{service: service}
// }

// // ==========================================
// // META
// // ==========================================

// // --- relationship handler Count ---

// type CountRes struct {
// 	Count int64 `json:"count"`
// }

// func (h *Handler) Count(w http.ResponseWriter, r *http.Request) error {
// 	count, err := h.service.Count(r.Context())
// 	if err != nil {
// 		return err
// 	}

// 	httpio.RespondOK(w, r, CountRes{Count: count}, "")
// 	return nil
// }

// // ==========================================
// // LIST
// // ==========================================

// // --- relationship handler List ---

// type ListQuery struct {
// 	Status Status `query:"status"`
// }

// func (h *Handler) List(w http.ResponseWriter, r *http.Request) error {
// 	userID, err := httpio.GetCtxUserID(r.Context())
// 	if err != nil {
// 		return apperr.NewUnauthorized(err, "")
// 	}

// 	query, err := httpio.BindQuery[ListQuery](r)
// 	if err != nil {
// 		return err
// 	}

// 	views, err := h.service.List(r.Context(), ListParams{
// 		UserID: userID,
// 		Status: query.Status,
// 	})
// 	if err != nil {
// 		return err
// 	}

// 	httpio.RespondOK(w, r, views, "")
// 	return nil
// }

// // ==========================================
// // UPSERT / UPDATE
// // ==========================================

// // --- relationship handler SendFriendRequestPath  ---

// type SendFriendRequestPath struct {
// 	ID uuid.UUID `path:"id" validate:"required"`
// }

// func (h *Handler) SendFriendRequest(w http.ResponseWriter, r *http.Request) error {
// 	userID, err := httpio.GetCtxUserID(r.Context())
// 	if err != nil {
// 		return apperr.NewUnauthorized(err, "")
// 	}

// 	path, err := httpio.BindPath[SendFriendRequestPath](r)
// 	if err != nil {
// 		return err
// 	}

// 	// Fixed: Changed TargetID to PeerID to match updated Service struct definitions
// 	if err := h.service.SendFriendRequest(r.Context(), SendFriendRequestParams{
// 		ActorID: userID,
// 		PeerID:  path.ID,
// 	}); err != nil {
// 		return err
// 	}

// 	httpio.RespondNoContent(w)
// 	return nil
// }

// // --- relationship handler AcceptFriendRequest  ---

// type AcceptFriendRequestPath struct {
// 	ID uuid.UUID `path:"id" validate:"required"`
// }

// func (h *Handler) AcceptFriendRequest(w http.ResponseWriter, r *http.Request) error {
// 	userID, err := httpio.GetCtxUserID(r.Context())
// 	if err != nil {
// 		return apperr.NewUnauthorized(err, "")
// 	}

// 	path, err := httpio.BindPath[AcceptFriendRequestPath](r)
// 	if err != nil {
// 		return err
// 	}

// 	// Fixed: Changed TargetID to PeerID to match updated Service struct definitions
// 	if err := h.service.AcceptFriendRequest(r.Context(), AcceptFriendRequestParams{
// 		ActorID: userID,
// 		PeerID:  path.ID,
// 	}); err != nil {
// 		return err
// 	}

// 	httpio.RespondNoContent(w)
// 	return nil
// }

// // --- relationship handler Block  ---

// type BlockPath struct {
// 	ID uuid.UUID `path:"id" validate:"required"`
// }

// func (h *Handler) Block(w http.ResponseWriter, r *http.Request) error {
// 	userID, err := httpio.GetCtxUserID(r.Context())
// 	if err != nil {
// 		return apperr.NewUnauthorized(err, "")
// 	}

// 	path, err := httpio.BindPath[BlockPath](r)
// 	if err != nil {
// 		return err
// 	}

// 	// Fixed: Changed TargetID to PeerID to match updated Service struct definitions
// 	if err := h.service.Block(r.Context(), BlockParams{
// 		ActorID: userID,
// 		PeerID:  path.ID,
// 	}); err != nil {
// 		return err
// 	}

// 	httpio.RespondNoContent(w)
// 	return nil
// }

// // ==========================================
// // DELETE
// // ==========================================

// // --- relationship handler Delete  ---

// type DeletePath struct {
// 	ID uuid.UUID `path:"id" validate:"required"`
// }

// func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) error {
// 	userID, err := httpio.GetCtxUserID(r.Context())
// 	if err != nil {
// 		return apperr.NewUnauthorized(err, "")
// 	}

// 	path, err := httpio.BindPath[DeletePath](r)
// 	if err != nil {
// 		return err
// 	}

// 	// Fixed: Changed TargetID to PeerID to match updated Service struct definitions
// 	if err := h.service.Delete(r.Context(), DeleteParams{
// 		ActorID: userID,
// 		PeerID:  path.ID,
// 	}); err != nil {
// 		return err
// 	}

// 	httpio.RespondNoContent(w)
// 	return nil
// }
