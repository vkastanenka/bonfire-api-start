package logger

import (
	"log/slog"
	"os"
)

// Config holds the settings for the wrapper.
type Config struct {
	Level     slog.Level
	AddSource bool
}

// Init configures the global slog instance.
// It wraps a slog JSON handler to inject request data into all log entries.
func Init(cfg Config) {
	// Create the JSON handler
	jsonHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     cfg.Level,
		AddSource: cfg.AddSource,
	})

	// Wrap with the context handler
	handler := NewHandler(jsonHandler)

	// Set new default
	slog.SetDefault(slog.New(handler))
}
