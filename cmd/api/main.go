package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"bonfire-api/internal/apperr"
	"bonfire-api/internal/auth"
	"bonfire-api/internal/config"
	"bonfire-api/internal/database"
	"bonfire-api/internal/email"
	"bonfire-api/internal/httpio"
	bfMiddleware "bonfire-api/internal/middleware"
	"bonfire-api/internal/repository"
	"bonfire-api/internal/validator"
	"bonfire-api/internal/worker"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-redis/redis_rate/v10"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"github.com/rs/cors"
)

func main() {
	initLogger()

	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, falling back to system environment variables")
	}

	// 1. Load Configuration
	cfg := config.Load()
	ctx := context.Background()

	// 2. Initialize PostgreSQL using your internal package
	// Assuming cfg.DatabaseURL is populated, otherwise use os.Getenv("DATABASE_URL")
	dbPool, err := database.NewPostgresPool(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v\n", err)
	}
	defer dbPool.Close()

	log.Println("Database connection pool established successfully")

	// Add to your main.go setup
	redisAddr := os.Getenv("REDIS_URL") // e.g., "localhost:6379"
	if redisAddr == "" {
		redisAddr = "localhost:6379" // Default for local
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	// Ping to ensure connectivity
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Could not connect to Redis: %v", err)
	}

	// Load JWT Secrets
	accessSecret := os.Getenv("JWT_ACCESS_SECRET")
	refreshSecret := os.Getenv("JWT_REFRESH_SECRET")
	verificationSecret := os.Getenv("JWT_VERIFICATION_SECRET")
	passwordResetSecret := os.Getenv("JWT_PASSWORD_RESET_SECRET")
	if accessSecret == "" || refreshSecret == "" {
		log.Fatal("JWT_ACCESS_SECRET and JWT_REFRESH_SECRET environment variables are required")
	}

	tokenConfig := auth.TokenConfig{
		AccessSecret:        accessSecret,
		RefreshSecret:       refreshSecret,
		VerificationSecret:  verificationSecret,
		PasswordResetSecret: passwordResetSecret,
	}

	// 1. Initialize storage and query wrappers
	store := repository.NewStore(dbPool) // For services running atomic transactions
	queries := repository.New(dbPool)    // For the background worker executing raw inline queries

	// 2. Initialize email infrastructure dependencies
	var mailer worker.Mailer
	resendAPIKey := os.Getenv("RESEND_API_KEY")

	if resendAPIKey != "" {
		// Production/Staging Mode
		fromAddress := os.Getenv("EMAIL_FROM_ADDRESS") // e.g., "Bonfire <noreply@yourdomain.com>"
		frontendURL := os.Getenv("FRONTEND_URL")       // e.g., "http://localhost:5173"
		overrideTo := os.Getenv("EMAIL_OVERRIDE_TO")   // NEW: Fetch the override variable

		if fromAddress == "" || frontendURL == "" {
			log.Fatal("EMAIL_FROM_ADDRESS and FRONTEND_URL are required when using Resend")
		}

		// UPDATE: Pass overrideTo as the 4th argument
		mailer = email.NewResendMailer(resendAPIKey, fromAddress, frontendURL, overrideTo)

		if overrideTo != "" {
			log.Printf("Email Engine: Resend initialized in SANDBOX mode (Overrides to: %s)", overrideTo)
		} else {
			log.Println("Email Engine: Resend initialized in PRODUCTION mode")
		}
	} else {
		// Local Development Mode
		mailer = email.NewLogMockMailer()
		log.Println("Email Engine: Mock Mailer initialized (console output only)")
	}

	// 3. Initialize Services and pass the store wrapper
	authService := auth.NewAuthService(store, tokenConfig)

	// 4. Instantiate and BOOT the background outbox processing engine
	// Polls the database every 1 second, processing up to 10 events per batch
	outboxWorker := worker.NewOutboxWorker(queries, mailer, 1*time.Second, 10)
	outboxWorker.Start()
	defer outboxWorker.Stop() // Guarantees the ticker stops cleanly if main terminates

	// Setup router
	r := chi.NewRouter()

	// Initialize Redis
	rateLimiter := redis_rate.NewLimiter(rdb)

	// 1. Initialize CORS
	corsMiddleware := cors.New(cors.Options{
		// Define your allowed origins (use your frontend domain in production)
		AllowedOrigins:   []string{"http://localhost:5173", "https://yourproductiondomain.com"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	})

	// Global Middlewares
	r.Use(corsMiddleware.Handler)
	r.Use(middleware.RequestID)
	r.Use(bfMiddleware.LoggingMiddleware) // <--- ADD THIS HERE
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(bfMiddleware.SecurityHeaders)

	// 4. Initialize validator and handler layer
	val := validator.New()
	authHandler := auth.NewAuthHandler(authService, val)

	r.Route("/api/v1", func(api chi.Router) {
		// ----------------------------------------------------
		// PUBLIC ROUTES (No token required)
		// ----------------------------------------------------
		// Strict limiting for Auth

		api.Group(func(auth chi.Router) {
			auth.Use(bfMiddleware.RateLimit(rateLimiter, 5, time.Minute, "auth"))
			api.Get("/auth/ping", httpio.ToHTTP(authHandler.Ping))
			api.Post("/auth/register", httpio.ToHTTP(authHandler.Register))
			api.Post("/auth/verify", httpio.ToHTTP(authHandler.VerifyEmail))
			api.Post("/auth/resend-verification-email", httpio.ToHTTP(authHandler.ResendVerificationEmail))
			api.Post("/auth/login", httpio.ToHTTP(authHandler.Login))
			api.Post("/auth/refresh", httpio.ToHTTP(authHandler.RefreshToken))
			api.Post("/auth/forgot-password", httpio.ToHTTP(authHandler.ForgotPassword))
			api.Post("/auth/reset-password", httpio.ToHTTP(authHandler.ResetPassword))
			api.Post("/auth/login/2fa", httpio.ToHTTP(authHandler.VerifyLogin2FA))
		})

		// ----------------------------------------------------
		// PROTECTED ROUTES (Valid Access Token required)
		// ----------------------------------------------------
		api.Group(func(protected chi.Router) {
			// Moderate limiting for general API
			protected.Use(bfMiddleware.RateLimit(rateLimiter, 100, time.Minute, "api"))
			// Apply the authorization middleware to EVERYTHING in this group
			protected.Use(auth.RequireAuth(accessSecret))

			protected.Get("/auth/devices", httpio.ToHTTP(authHandler.GetDevices))
			protected.Delete("/auth/devices", httpio.ToHTTP(authHandler.RevokeAllOtherDevices)) // Deletes all others
			protected.Delete("/auth/devices/{id}", httpio.ToHTTP(authHandler.RevokeDevice))     // Deletes specific

			// 2. Strict Verification Sub-Group
			protected.Group(func(verified chi.Router) {
				// Only verified users pass this line
				verified.Use(auth.RequireVerified())

				// NEW: 2FA Setup (User must be logged in AND have a verified email)
				verified.Post("/users/me/2fa/generate", httpio.ToHTTP(authHandler.GenerateTOTP))
				verified.Post("/users/me/2fa/enable", httpio.ToHTTP(authHandler.EnableTOTP))

				// // Unverified users CANNOT access these (they get 403 Forbidden):
				// verified.Post("/guilds", httpio.ToHTTP(guildHandler.CreateGuild))
				// verified.Post("/messages", httpio.ToHTTP(messageHandler.SendMessage))
			})
		})
	})

	// Chi-idiomatic global fallback for missing endpoints (404)
	r.NotFound(httpio.ToHTTP(func(w http.ResponseWriter, r *http.Request) error {
		return apperr.NewNotFound("The requested API endpoint does not exist.")
	}))

	// Chi-idiomatic global fallback for bad verbs (405)
	r.MethodNotAllowed(httpio.ToHTTP(func(w http.ResponseWriter, r *http.Request) error {
		return apperr.NewMethodNotAllowed("HTTP method not allowed for this endpoint.")
	}))

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	log.Println("Core API Server starting on :8080...")
	log.Fatal(srv.ListenAndServe())
}

func initLogger() {
	// Structured JSON logging is industry standard for aggregation
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo, // Use LevelDebug during local dev
	})
	logger := slog.New(handler)
	slog.SetDefault(logger)
}

// Logged-in Devices (Login)
