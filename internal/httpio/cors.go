package httpio

import (
	"net/http"

	"bonfire-api/internal/config"

	"github.com/rs/cors"
)

// CORS constructs cross-origin resource sharing middleware based on application configuration.
func CORS(cfg *config.Config) func(http.Handler) http.Handler {
	c := cors.New(cors.Options{
		AllowedOrigins: cfg.CORSAllowedOrigins,
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowedHeaders: []string{
			"Accept",
			"Authorization",
			"Content-Type",
			"X-CSRF-Token",
			"X-Requested-With",
			"X-Trace-ID",
		},
		ExposedHeaders: []string{
			"Link",
			"X-Request-ID",
			"X-Trace-ID",
		},
		AllowCredentials: true,
		MaxAge:           300,
	})

	return c.Handler
}
