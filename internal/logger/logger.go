package logger

import (
	"log/slog"
	"os"
)

// initLogger configures the global slog instance. It wraps a standard JSON
// handler with custom middleware to automatically inject request-scoped data
// (such as trace IDs) into all log entries.
func InitLogger() {
	// Create the JSON handler
	jsonHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: true, // Set to true only for local debugging
	})

	// Wrap with the context handler
	handler := NewContextHandler(jsonHandler)

	// Set new default
	slog.SetDefault(slog.New(handler))
}
