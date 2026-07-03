package httpio

import (
	"net/http"

	"bonfire-api/internal/config"

	"github.com/rs/cors"
)

func CORS(cfg *config.Config) func(http.Handler) http.Handler {
	c := cors.New(cors.Options{
		AllowedOrigins: cfg.CORSAllowedOrigins,
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowedHeaders: []string{
			"Accept",
			"Authorization",
			"Content-Type",
			"X-CSRF-Token",
		},
		ExposedHeaders: []string{
			"Link",
			"X-Request-ID",
			"X-Trace-ID",
		},
		AllowCredentials: cfg.CORSAllowCredentials,
		MaxAge:           300,
	})

	return c.Handler
}
