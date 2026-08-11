package main

import (
	"net/http"

	"bonfire-api/internal/errs"
	"bonfire-api/internal/httpio"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func (app *Application) routes() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(httpio.Trace)
	r.Use(httpio.WithClientMeta(app.Config.TrustProxy))
	r.Use(httpio.Logger)
	r.Use(httpio.Recoverer)
	r.Use(httpio.CORS(app.Config))
	r.Use(middleware.Timeout(app.Config.RequestTimeout))
	r.Use(httpio.SecurityHeaders)

	r.Route("/api/v1", func(api chi.Router) {
		api.Group(func(public chi.Router) {
			public.Use(httpio.RateLimit(app.RateLimiter, httpio.RateLimitConfig{
				Limit:  app.Config.AuthRateLimit,
				Window: app.Config.AuthRateWindow,
				Scope:  httpio.RateLimitScopePublic,
			}))

			public.Get("/healthz", httpio.ToHTTPErr(app.Handlers.Health.Check))

			public.Post("/auth/register", httpio.ToHTTPErr(app.Handlers.Auth.Register))
			public.Post("/auth/login", httpio.ToHTTPErr(app.Handlers.Auth.Login))
			public.Post("/auth/refresh", httpio.ToHTTPErr(app.Handlers.Auth.Refresh))
			public.Post("/auth/verify", httpio.ToHTTPErr(app.Handlers.Auth.VerifyEmail))
			public.Post("/auth/forgot-password", httpio.ToHTTPErr(app.Handlers.Auth.ForgotPassword))
			public.Post("/auth/reset-password", httpio.ToHTTPErr(app.Handlers.Auth.ResetPassword))

			public.Get("/gateway/ws", httpio.ToHTTPErr(app.Handlers.Gateway.ServeWS))

			public.Get("/users/{id}", httpio.ToHTTPErr(app.Handlers.User.Get))
		})

		api.Group(func(auth chi.Router) {
			auth.Use(httpio.RateLimit(app.RateLimiter, httpio.RateLimitConfig{
				Limit:  app.Config.AuthRateLimit,
				Window: app.Config.AuthRateWindow,
				Scope:  httpio.RateLimitScopeAuth,
			}))
			auth.Use(httpio.RequireAuth(app.Tokens))

			auth.Post("/auth/resend-verify", httpio.ToHTTPErr(app.Handlers.Auth.ResendVerify))
			auth.Post("/auth/ws-ticket", httpio.ToHTTPErr(app.Handlers.Auth.WSTicket))

			// Account & Profile Management
			auth.Get("/users/@me", httpio.ToHTTPErr(app.Handlers.Me.Get))
			auth.Patch("/users/@me/email", httpio.ToHTTPErr(app.Handlers.Me.UpdateEmail))
			auth.Patch("/users/@me/username", httpio.ToHTTPErr(app.Handlers.Me.UpdateUsername))
			auth.Post("/users/@me/password", httpio.ToHTTPErr(app.Handlers.Me.UpdatePassword))
			auth.Patch("/users/@me/presence", httpio.ToHTTPErr(app.Handlers.Me.UpdatePreferredPresence))
			auth.Patch("/users/@me/profile", httpio.ToHTTPErr(app.Handlers.Me.UpdateProfile))
			auth.Post("/users/@me/disable", httpio.ToHTTPErr(app.Handlers.Me.Disable))
			auth.Delete("/users/@me", httpio.ToHTTPErr(app.Handlers.Me.ScheduleDelete))

			auth.Post("/users/@me/relationships/{id}", httpio.ToHTTPErr(app.Handlers.Me.SendFriendRequest))
			auth.Post("/users/@me/relationships/{id}/accept", httpio.ToHTTPErr(app.Handlers.Me.AcceptFriendRequest))
			auth.Post("/users/@me/relationships/{id}/block", httpio.ToHTTPErr(app.Handlers.Me.Block))
			auth.Delete("/users/@me/relationships/{id}", httpio.ToHTTPErr(app.Handlers.Me.RemoveRelationship))

			auth.Post("/channels", httpio.ToHTTPErr(app.Handlers.Channel.Create))
			// auth.Get("/channels/{channelId}", httpio.ToHTTPErr(app.Handlers.Channel.Get))
			auth.Get("/channels/{channelId}/messages", httpio.ToHTTPErr(app.Handlers.Channel.ListMessages))
			auth.Post("/channels/{channelId}/messages", httpio.ToHTTPErr(app.Handlers.Channel.PostMessage))
			auth.Patch("/channels/{channelId}/messages/{messageId}", httpio.ToHTTPErr(app.Handlers.Channel.EditMessage))
			auth.Delete("/channels/{channelId}/messages/{messageId}", httpio.ToHTTPErr(app.Handlers.Channel.DeleteMessage))
		})
	})

	r.NotFound(httpio.ToHTTPErr(app.notFoundHandler))
	r.MethodNotAllowed(httpio.ToHTTPErr(app.methodNotAllowedHandler))

	return r
}

func (app *Application) notFoundHandler(w http.ResponseWriter, r *http.Request) error {
	return errs.NotFound("The requested API endpoint does not exist.")
}

func (app *Application) methodNotAllowedHandler(w http.ResponseWriter, r *http.Request) error {
	return errs.Unimplemented("HTTP method not allowed for this endpoint")
}
