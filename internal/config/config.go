package config

import (
	"log"
	"os"
)

type Config struct {
	Port        string
	DatabaseURL string
	RedisURL    string
	// Add your JWT secrets and Resend keys here...
}

func Load() *Config {
	// Note: Call godotenv.Load() in main.go before calling this
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	return &Config{
		Port:        ":8080", // Could also be loaded from env
		DatabaseURL: dbURL,
		RedisURL:    os.Getenv("REDIS_URL"),
		// Populate the rest...
	}
}
