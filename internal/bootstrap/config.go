package bootstrap

import (
	"bonfire-api/internal/config"
	"log/slog"

	"github.com/joho/godotenv"
)

// InitConfig handles environment and config loading
func InitConfig() (*config.Config, error) {
	// Init env
	godotenv.Load()

	// Load config
	cfg, err := config.Load()
	if err != nil {
		slog.Error("configuration load failed", "error", err)
		return nil, err
	}

	// Finish
	slog.Info("configuration loaded successfully", "environment", cfg.AppEnv)
	return cfg, nil
}
