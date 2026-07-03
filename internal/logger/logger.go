package logger

import (
	"log/slog"
	"os"
)

type Config struct {
	Level     slog.Level
	AddSource bool
}

func Init(cfg Config) {
	jsonHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     cfg.Level,
		AddSource: cfg.AddSource,
	})

	handler := NewHandler(jsonHandler)

	slog.SetDefault(slog.New(handler))
}
