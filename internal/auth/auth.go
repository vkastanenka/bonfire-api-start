package auth

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

type Service struct {
	// store       repository.Store
	// cache       cache.Store
	// token       *token.Manager
	// session     *session.Service
	// user        *user.Service
	// flightGroup singleflight.Group
}

func NewService(
// store repository.Store,
// cache cache.Store,
// token *token.Manager,
// session *session.Service,
// user *user.Service,
) *Service {
	return &Service{
		// store:       store,
		// cache:       cache,
		// token:       token,
		// session:     session,
		// user:        user,
		// flightGroup: singleflight.Group{},
	}
}
