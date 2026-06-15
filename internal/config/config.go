package config

import (
	"log"
	"os"
)

type Config struct {
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

func Load() *Config {
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
		log.Fatal("DATABASE_URL is required")
	}
	if cacheURL == "" {
		log.Fatal("REDIS_URL is required")
	}
	if accessSecret == "" {
		log.Fatal("JWT_ACCESS_SECRET is required")
	}
	if refreshSecret == "" {
		log.Fatal("JWT_REFRESH_SECRET is required")
	}
	if verificationSecret == "" {
		log.Fatal("JWT_VERIFICATION_SECRET is required")
	}
	if passwordResetSecret == "" {
		log.Fatal("JWT_PASSWORD_RESET_SECRET is required")
	}

	// 3. Conditional Email Validation (Allows local mock fallback)
	if resendApiKey != "" {
		if emailFromAddress == "" || frontendURL == "" {
			log.Fatal("EMAIL_FROM_ADDRESS and FRONTEND_URL are required when using Resend")
		}
	}

	return &Config{
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
	}
}
