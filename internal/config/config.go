package config

import (
	"bonfire-api/internal/auth"
	"errors"
	"os"
)

type Config struct {
	AppEnv              string
	Port                string
	DatabaseURL         string
	RedisURL            string
	AccessSecret        string
	RefreshSecret       string
	VerificationSecret  string
	PasswordResetSecret string
	ResendApiKey        string
	EmailFromAddress    string
	FrontendURL         string
	EmailOverrideTo     string
}

func Load() (*Config, error) {
	// 1. Fetch all variables
	dbURL := os.Getenv("DATABASE_URL")
	cacheURL := os.Getenv("REDIS_URL")
	accessSecret := os.Getenv("JWT_ACCESS_SECRET")
	refreshSecret := os.Getenv("JWT_REFRESH_SECRET")
	verificationSecret := os.Getenv("JWT_VERIFICATION_SECRET")
	passwordResetSecret := os.Getenv("JWT_PASSWORD_RESET_SECRET")
	resendApiKey := os.Getenv("RESEND_API_KEY")
	emailFromAddress := os.Getenv("EMAIL_FROM_ADDRESS")
	frontendURL := os.Getenv("FRONTEND_URL")
	emailOverrideTo := os.Getenv("EMAIL_OVERRIDE_TO")

	// 2. Strict Core Validations
	if dbURL == "" {
		return nil, errors.New("DATABASE_URL is required")
	}
	if cacheURL == "" {
		return nil, errors.New("REDIS_URL is required")
	}
	if accessSecret == "" {
		return nil, errors.New("JWT_ACCESS_SECRET is required")
	}
	if refreshSecret == "" {
		return nil, errors.New("JWT_REFRESH_SECRET is required")
	}
	if verificationSecret == "" {
		return nil, errors.New("JWT_VERIFICATION_SECRET is required")
	}
	if passwordResetSecret == "" {
		return nil, errors.New("JWT_PASSWORD_RESET_SECRET is required")
	}

	// 3. Conditional Email Validation (Allows local mock fallback)
	if resendApiKey != "" {
		if emailFromAddress == "" || frontendURL == "" {
			return nil, errors.New("EMAIL_FROM_ADDRESS and FRONTEND_URL are required when using Resend")
		}
	}

	return &Config{
		AppEnv:              "development",
		Port:                ":8080",
		DatabaseURL:         dbURL,
		RedisURL:            cacheURL,
		AccessSecret:        accessSecret,
		RefreshSecret:       refreshSecret,
		VerificationSecret:  verificationSecret,
		PasswordResetSecret: passwordResetSecret,
		ResendApiKey:        resendApiKey,
		EmailFromAddress:    emailFromAddress,
		FrontendURL:         frontendURL,
		EmailOverrideTo:     emailOverrideTo,
	}, nil
}

func (c *Config) TokenConfig() auth.TokenConfig {
	return auth.TokenConfig{
		AccessSecret:        c.AccessSecret,
		RefreshSecret:       c.RefreshSecret,
		VerificationSecret:  c.VerificationSecret,
		PasswordResetSecret: c.PasswordResetSecret,
	}
}
