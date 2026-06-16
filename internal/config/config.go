package config

import (
	"errors"
	"fmt"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

// Config holds all application configuration variables.
// Struct tags define the environment variable mapping, requirements, and defaults.
type Config struct {
	AppEnv               string `env:"APP_ENV" envDefault:"development"`
	Port                 string `env:"PORT" envDefault:":8080"`
	DatabaseURL          string `env:"DATABASE_URL,required"`
	RedisURL             string `env:"REDIS_URL,required"`
	AccessSecret         string `env:"JWT_ACCESS_SECRET,required"`
	RefreshSecret        string `env:"JWT_REFRESH_SECRET,required"`
	VerificationSecret   string `env:"JWT_VERIFICATION_SECRET,required"`
	PasswordResetSecret  string `env:"JWT_PASSWORD_RESET_SECRET,required"`
	ResendApiKey         string `env:"RESEND_API_KEY"`
	EmailFromAddress     string `env:"EMAIL_FROM_ADDRESS"`
	FrontendURL          string `env:"FRONTEND_URL"`
	EmailOverrideTo      string `env:"EMAIL_OVERRIDE_TO"`
	CORSAllowedOrigins   string `env:"CORS_ALLOWED_ORIGINS" envDefault:"http://localhost:5173"`
	CORSAllowCredentials bool   `env:"CORS_ALLOW_CREDENTIALS" envDefault:"true"`
}

// Load parses environment variables into the Config struct.
func Load() (*Config, error) {
	// Load env
	godotenv.Load()

	var cfg Config

	// Parse env
	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("configuration error: %w", err)
	}

	// Complex validation
	if cfg.ResendApiKey != "" {
		if cfg.EmailFromAddress == "" || cfg.FrontendURL == "" {
			return nil, errors.New("EMAIL_FROM_ADDRESS and FRONTEND_URL are required when using Resend")
		}
	}

	return &cfg, nil
}
