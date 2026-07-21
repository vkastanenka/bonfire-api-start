package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv               string        `env:"APP_ENV" envDefault:"development"`
	Port                 string        `env:"PORT" envDefault:":8080"`
	TrustProxy           bool          `env:"TRUST_PROXY" envDefault:"false"`
	TokenIssuer          string        `env:"TOKEN_ISSUER" envDefault:"bonfire-api"`
	AccessSecret         string        `env:"JWT_ACCESS_SECRET,required"`
	RefreshSecret        string        `env:"JWT_REFRESH_SECRET,required"`
	EmailVerifySecret    string        `env:"JWT_EMAIL_VERIFY_SECRET,required"`
	PasswordResetSecret  string        `env:"JWT_PASSWORD_RESET_SECRET,required"`
	PasswordMFASecret    string        `env:"JWT_PASSWORD_MFA_SECRET,required"`
	ResendApiKey         string        `env:"RESEND_API_KEY"`
	EmailFromAddress     string        `env:"EMAIL_FROM_ADDRESS"`
	FrontendURL          string        `env:"FRONTEND_URL"`
	EmailOverrideTo      string        `env:"EMAIL_OVERRIDE_TO"`
	RequestTimeout       time.Duration `env:"HTTP_REQUEST_TIMEOUT" envDefault:"10s"`
	ServerWriteTimeout   time.Duration `env:"HTTP_WRITE_TIMEOUT" envDefault:"20s"`
	ServerReadTimeout    time.Duration `env:"HTTP_READ_TIMEOUT" envDefault:"5s"`
	ShutdownTimeout      time.Duration `env:"HTTP_SHUTDOWN_TIMEOUT" envDefault:"5s"`
	AuthRateLimit        int           `env:"AUTH_RATE_LIMIT" envDefault:"5"`
	AuthRateWindow       time.Duration `env:"AUTH_RATE_WINDOW" envDefault:"1m"`
	CORSAllowedOrigins   []string      `env:"CORS_ALLOWED_ORIGINS" envDefault:"http://localhost:5173"`
	CORSAllowCredentials bool          `env:"CORS_ALLOW_CREDENTIALS" envDefault:"true"`
	DatabaseURL          string        `env:"DATABASE_URL,required"`
	DBMaxConns           int32         `env:"DB_MAX_CONNS" envDefault:"25"`
	DBMinConns           int32         `env:"DB_MIN_CONNS" envDefault:"2"`
	DBMaxConnLifetime    time.Duration `env:"DB_MAX_CONN_LIFETIME" envDefault:"1h"`
	DBMaxConnIdleTime    time.Duration `env:"DB_MAX_CONN_IDLE_TIME" envDefault:"30m"`
	DBHealthCheck        time.Duration `env:"DB_HEALTH_CHECK" envDefault:"1m"`
	RedisURL             string        `env:"REDIS_URL,required"`
	RedisPoolSize        int           `env:"REDIS_POOL_SIZE" envDefault:"20"`
	RedisMinIdleConns    int           `env:"REDIS_MIN_IDLE_CONNS" envDefault:"2"`
	RedisConnMaxIdleTime time.Duration `env:"REDIS_CONN_MAX_IDLE_TIME" envDefault:"30m"`
	RedisConnMaxLifetime time.Duration `env:"REDIS_CONN_MAX_LIFETIME" envDefault:"1h"`
	PresenceTTL          time.Duration `env:"PRESENCE_TTL" envDefault:"30s"`
}

func (c *Config) IsDevelopment() bool {
	return c.AppEnv == "development"
}

func (c *Config) IsStaging() bool {
	return c.AppEnv == "staging"
}

func (c *Config) IsProduction() bool {
	return c.AppEnv == "production"
}

func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load .env file: %w", err)
	}

	var cfg Config

	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("configuration parsing error: %w", err)
	}

	cfg.normalizePort()
	cfg.normalizeEnv()

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("configuration validation error: %w", err)
	}

	return &cfg, nil
}

func (c *Config) validate() error {
	if c.ResendApiKey != "" {
		if c.EmailFromAddress == "" || c.FrontendURL == "" {
			return errors.New("EMAIL_FROM_ADDRESS and FRONTEND_URL are required when RESEND_API_KEY is provided")
		}
	}
	return nil
}

func (c *Config) normalizePort() {
	c.Port = strings.TrimSpace(c.Port)
	if c.Port != "" && !strings.HasPrefix(c.Port, ":") {
		c.Port = ":" + c.Port
	}
}

func (c *Config) normalizeEnv() {
	c.AppEnv = strings.ToLower(strings.TrimSpace(c.AppEnv))
	if c.AppEnv == "" {
		c.AppEnv = "development"
	}
}
