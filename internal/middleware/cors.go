package middleware

import (
	"bonfire-api/internal/config"
	"net/http"

	"github.com/rs/cors"
)

func Cors(cfg *config.Config) func(http.Handler) http.Handler {
	c := cors.New(cors.Options{
		AllowedOrigins: cfg.CORSAllowedOrigins,
		AllowedMethods: []string{
			http.MethodGet, http.MethodPost, http.MethodPut,
			http.MethodDelete, http.MethodOptions,
		},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: cfg.CORSAllowCredentials,
		MaxAge:           300,
	})

	return c.Handler
}
