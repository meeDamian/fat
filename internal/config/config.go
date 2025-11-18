package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/lmittmann/tint"
	"golang.org/x/term"
)

type Config struct {
	ServerAddress       string
	ModelRequestTimeout time.Duration
	LogLevel            string
}

func Load() (Config, error) {
	cfg := Config{
		ServerAddress:       envOrDefault("FAT_SERVER_ADDR", ":4444"),
		ModelRequestTimeout: 120 * time.Second, // Increased to 120s for GPT-5 models
		LogLevel:            envOrDefault("FAT_LOG_LEVEL", "info"),
	}

	if timeoutStr := os.Getenv("FAT_MODEL_TIMEOUT"); timeoutStr != "" {
		duration, err := time.ParseDuration(timeoutStr)
		if err != nil {
			return Config{}, fmt.Errorf("invalid FAT_MODEL_TIMEOUT value %q: %w", timeoutStr, err)
		}
		cfg.ModelRequestTimeout = duration
	}

	return cfg, nil
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func NewLogger(level string) (*slog.Logger, error) {
	var slogLevel slog.Level
	switch strings.ToLower(level) {
	case "debug":
		slogLevel = slog.LevelDebug
	case "warn":
		slogLevel = slog.LevelWarn
	case "error":
		slogLevel = slog.LevelError
	case "info", "":
		slogLevel = slog.LevelInfo
	default:
		return nil, fmt.Errorf("unknown log level %q", level)
	}

	// Use beautiful colored output for terminal, JSON for pipes/files
	var handler slog.Handler
	if term.IsTerminal(int(os.Stdout.Fd())) {
		// Terminal: use tint for beautiful colored output
		handler = tint.NewHandler(os.Stdout, &tint.Options{
			Level:      slogLevel,
			TimeFormat: "15:04", // 24-hour format
			AddSource:  slogLevel == slog.LevelDebug,
		})
	} else {
		// Non-terminal (pipe, file, production): use JSON
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slogLevel,
		})
	}

	return slog.New(handler), nil
}
