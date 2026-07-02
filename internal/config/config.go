package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

// Config holds all application configuration variables.
type Config struct {
	AppEnv               string   `env:"APP_ENV" envDefault:"development"`
	Port                 string   `env:"PORT" envDefault:":8080"`
	DatabaseURL          string   `env:"DATABASE_URL,required"`
	RedisURL             string   `env:"REDIS_URL,required"`
	AccessSecret         string   `env:"JWT_ACCESS_SECRET,required"`
	RefreshSecret        string   `env:"JWT_REFRESH_SECRET,required"`
	VerificationSecret   string   `env:"JWT_VERIFICATION_SECRET,required"`
	PasswordResetSecret  string   `env:"JWT_PASSWORD_RESET_SECRET,required"`
	PasswordMFASecret    string   `env:"JWT_PASSWORD_MFA_SECRET,required"`
	ResendApiKey         string   `env:"RESEND_API_KEY"`
	EmailFromAddress     string   `env:"EMAIL_FROM_ADDRESS"`
	FrontendURL          string   `env:"FRONTEND_URL"`
	EmailOverrideTo      string   `env:"EMAIL_OVERRIDE_TO"`
	CORSAllowedOrigins   []string `env:"CORS_ALLOWED_ORIGINS" envDefault:"http://localhost:5173"`
	CORSAllowCredentials bool     `env:"CORS_ALLOW_CREDENTIALS" envDefault:"true"`
}

// IsDevelopment returns true if the application is running in development.
func (c *Config) IsDevelopment() bool {
	return c.AppEnv == "development"
}

// IsStaging returns true if the application is running in staging.
func (c *Config) IsStaging() bool {
	return c.AppEnv == "staging"
}

// IsProduction returns true if the application is running in production.
func (c *Config) IsProduction() bool {
	return c.AppEnv == "production"
}

// Load parses and validates environment variables into the Config struct.
func Load() (*Config, error) {
	// Attempt to load .env file; ignore failure if it doesn't exist
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load .env file: %w", err)
	}

	var cfg Config

	// Parse environment variables into struct
	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("configuration parsing error: %w", err)
	}

	// Sanitize
	cfg.normalizePort()
	cfg.normalizeEnv()

	// Validate
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("configuration validation error: %w", err)
	}

	return &cfg, nil
}

// validate enforces business constraints.
func (c *Config) validate() error {
	if c.ResendApiKey != "" {
		if c.EmailFromAddress == "" || c.FrontendURL == "" {
			return errors.New("EMAIL_FROM_ADDRESS and FRONTEND_URL are required when RESEND_API_KEY is provided")
		}
	}
	return nil
}

// normalizePort ensures the port always starts with a colon, handling both "8080" and ":8080"
func (c *Config) normalizePort() {
	c.Port = strings.TrimSpace(c.Port)
	if c.Port != "" && !strings.HasPrefix(c.Port, ":") {
		c.Port = ":" + c.Port
	}
}

// normalizeEnv normalizes the environment string once at boot time
func (c *Config) normalizeEnv() {
	c.AppEnv = strings.ToLower(strings.TrimSpace(c.AppEnv))
	if c.AppEnv == "" {
		c.AppEnv = "development"
	}
}

// String prevent leaks of sensitive credentials if the configuration is ever printed to stdout/logs.
func (c *Config) String() string {
	return fmt.Sprintf(
		"AppEnv: %s | Port: %s | DatabaseURL: [REDACTED] | RedisURL: [REDACTED] | CORSAllowedOrigins: %v",
		c.AppEnv, c.Port, c.CORSAllowedOrigins,
	)
}
