package logger

import (
	"log/slog"
	"os"
)

// Config holds initialization settings for the global logger.
type Config struct {
	Level     slog.Level
	AddSource bool
}

// Init configures and sets the default global slog logger using JSON output.
func Init(cfg Config) {
	jsonHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     cfg.Level,
		AddSource: cfg.AddSource,
	})
	handler := NewHandler(jsonHandler)
	slog.SetDefault(slog.New(handler))
}
