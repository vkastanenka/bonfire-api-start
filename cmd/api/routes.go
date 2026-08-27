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

		// Public Routes (Rate limited by IP/Public scope)
		api.Group(func(public chi.Router) {
			public.Use(httpio.RateLimit(app.RateLimiter, httpio.RateLimitConfig{
				Limit:  app.Config.AuthRateLimit,
				Window: app.Config.AuthRateWindow,
				Scope:  httpio.RateLimitScopePublic,
			}))

			public.Get("/healthz", httpio.ToHTTPErr(app.Handlers.Health.Check))

			// Auth (Public)
			public.Post("/auth/register", httpio.ToHTTPErr(app.Handlers.Auth.Register))
			public.Post("/auth/login", httpio.ToHTTPErr(app.Handlers.Auth.Login))
			public.Post("/auth/refresh", httpio.ToHTTPErr(app.Handlers.Auth.Refresh))
			public.Post("/auth/verify", httpio.ToHTTPErr(app.Handlers.Auth.VerifyEmail))
			public.Post("/auth/forgot-password", httpio.ToHTTPErr(app.Handlers.Auth.ForgotPassword))
			public.Post("/auth/reset-password", httpio.ToHTTPErr(app.Handlers.Auth.ResetPassword))

			// Realtime Gateway WS Handshake
			public.Get("/gateway/ws", httpio.ToHTTPErr(app.Handlers.Gateway.ServeWS))

		})

		// Authenticated Routes (Requires Valid Bearer Token)
		api.Group(func(auth chi.Router) {
			auth.Use(httpio.RateLimit(app.RateLimiter, httpio.RateLimitConfig{
				Limit:  app.Config.AuthRateLimit,
				Window: app.Config.AuthRateWindow,
				Scope:  httpio.RateLimitScopeAuth,
			}))
			auth.Use(httpio.RequireAuth(app.Tokens))

			// Auth (Private)
			auth.Post("/auth/resend-verify", httpio.ToHTTPErr(app.Handlers.Auth.ResendVerify))
			auth.Post("/auth/ws-ticket", httpio.ToHTTPErr(app.Handlers.Auth.PrintWSTicket))

			// User Profiles (Authenticated)
			auth.Get("/users/{userId}", httpio.ToHTTPErr(app.Handlers.User.Get))

			// Current User (@me)
			auth.Route("/users/@me", func(me chi.Router) {
				me.Get("/", httpio.ToHTTPErr(app.Handlers.User.GetMe))
				me.Patch("/email", httpio.ToHTTPErr(app.Handlers.User.UpdateEmail))
				me.Patch("/username", httpio.ToHTTPErr(app.Handlers.User.UpdateUsername))
				me.Post("/password", httpio.ToHTTPErr(app.Handlers.User.UpdatePassword))
				me.Patch("/presence", httpio.ToHTTPErr(app.Handlers.User.UpdatePreferredPresence))
				me.Patch("/profile", httpio.ToHTTPErr(app.Handlers.User.UpdateProfile))
				me.Post("/disable", httpio.ToHTTPErr(app.Handlers.User.Disable))
				me.Delete("/", httpio.ToHTTPErr(app.Handlers.User.ScheduleDelete))

				// User Sessions
				me.Get("/sessions", httpio.ToHTTPErr(app.Handlers.Session.ListValid))
				me.Delete("/sessions", httpio.ToHTTPErr(app.Handlers.Session.RevokeAll))
				me.Delete("/sessions/{sessionId}", httpio.ToHTTPErr(app.Handlers.Session.Revoke))

				// User Relationships & Social Graph
				me.Route("/relationships", func(rel chi.Router) {
					rel.Get("/friends", httpio.ToHTTPErr(app.Handlers.Relation.GetFriends))
					rel.Get("/pending", httpio.ToHTTPErr(app.Handlers.Relation.GetPending))
					rel.Get("/blocked", httpio.ToHTTPErr(app.Handlers.Relation.GetBlocked))
					rel.Get("/{peerId}", httpio.ToHTTPErr(app.Handlers.Relation.GetPeer))

					rel.Post("/{peerId}", httpio.ToHTTPErr(app.Handlers.Relation.SendRequest))
					rel.Post("/{peerId}/accept", httpio.ToHTTPErr(app.Handlers.Relation.AcceptRequest))
					rel.Post("/{peerId}/block", httpio.ToHTTPErr(app.Handlers.Relation.BlockUser))
					rel.Delete("/{peerId}", httpio.ToHTTPErr(app.Handlers.Relation.RemoveRelation))
				})
			})

			// Channels & Sub-resources
			auth.Route("/channels", func(ch chi.Router) {
				ch.Post("/group", httpio.ToHTTPErr(app.Handlers.Channel.CreateGroup))

				ch.Route("/{channelId}", func(c chi.Router) {
					c.Get("/", httpio.ToHTTPErr(app.Handlers.Channel.Get))
					c.Patch("/", httpio.ToHTTPErr(app.Handlers.Channel.UpdateGroup))

					// Channel Members & Direct Messages Management
					c.Post("/members", httpio.ToHTTPErr(app.Handlers.Member.AddMembers))
					c.Patch("/members/last-read", httpio.ToHTTPErr(app.Handlers.Member.UpdateLastReadMessage))
					c.Patch("/members/pinned", httpio.ToHTTPErr(app.Handlers.Member.UpdatePinnedAt))
					c.Patch("/members/muted", httpio.ToHTTPErr(app.Handlers.Member.UpdateMutedUntil))
					c.Delete("/members/leave", httpio.ToHTTPErr(app.Handlers.Member.LeaveGroup))
					c.Delete("/members/close", httpio.ToHTTPErr(app.Handlers.Member.CloseDirect))

					// Channel Messages
					c.Route("/messages", func(msg chi.Router) {
						msg.Get("/", httpio.ToHTTPErr(app.Handlers.Message.List))
						msg.Post("/", httpio.ToHTTPErr(app.Handlers.Message.Create))
						msg.Get("/pinned", httpio.ToHTTPErr(app.Handlers.Message.ListPinned))

						msg.Route("/{messageId}", func(m chi.Router) {
							m.Patch("/", httpio.ToHTTPErr(app.Handlers.Message.UpdateContent))
							m.Delete("/", httpio.ToHTTPErr(app.Handlers.Message.Delete))
							m.Patch("/pinned", httpio.ToHTTPErr(app.Handlers.Message.UpdatePinnedAt))
							m.Post("/reactions", httpio.ToHTTPErr(app.Handlers.Message.ToggleReaction))
						})
					})
				})
			})
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
